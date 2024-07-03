package app

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/idr0id/keenctl/internal/keenetic"
	"github.com/idr0id/keenctl/internal/resolve"
	"golang.org/x/sync/errgroup"
)

const (
	maxRetryDelay = 5 * time.Second
	defaultMinTTL = time.Hour
)

type App struct {
	conf         Config
	logger       *slog.Logger
	router       *keenetic.Router
	resolver     *resolve.Resolver
	resolveQueue *resolveQueue

	eg     errgroup.Group
	doneCh chan struct{}
}

func New(conf Config, logger *slog.Logger) *App {
	return &App{
		conf:     conf,
		logger:   logger,
		resolver: resolve.New(conf.Resolver, logger),
		doneCh:   make(chan struct{}),
	}
}

func (a *App) Run() error {
	var attempt int

	reconnectTimer := time.NewTimer(0)
	defer reconnectTimer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-a.doneCh:
			return nil

		case <-reconnectTimer.C:
			if attempt == 0 {
				a.logger.Info("connecting to router")
			}

			if err := a.connectToRouter(); err != nil {
				a.logger.Error(
					"connection to router failed",
					slog.Any("error", err),
					slog.Int("attempt", attempt),
				)
				attempt++

				delay := time.Duration(attempt) * time.Second
				if delay > maxRetryDelay {
					delay = maxRetryDelay
				}
				reconnectTimer.Reset(delay)
				continue
			}

			attempt = 0

			if err := a.resolveAndSync(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					a.logger.Error("syncing to router failed", slog.Any("error", err))
				}
			}
		}
	}
}

func (a *App) Shutdown() {
	close(a.doneCh)
	_ = a.eg.Wait()
}

func (a *App) resolveAndSync(ctx context.Context) error {
	routesCh := make(chan []keenetic.IPRoute, 1)

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-a.doneCh
		cancel()
	}()

	a.eg.Go(func() error {
		t := time.NewTimer(0)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				a.logger.Info("resolving addresses for routes")

				routes, expireAt, err := a.resolveRoutes(ctx)
				if err != nil {
					a.logger.Error(
						"resolving addresses for routes failed",
						slog.Any("error", err),
					)
					continue
				}

				routesCh <- routes

				a.logger.Debug(
					"scheduling resolving expired addresses for routes",
					slog.Time("expire_at", expireAt),
				)
				t.Reset(time.Until(expireAt))

			case <-ctx.Done():
				close(routesCh)
				return nil
			}
		}
	})

	a.eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil

			case routes := <-routesCh:
				if err := a.syncToRouter(ctx, routes); err != nil {
					cancel()
					return err
				}
			}
		}
	})

	return a.eg.Wait()
}

func (a *App) connectToRouter() error {
	router, err := keenetic.New(a.conf.SSH, a.logger.WithGroup("keenetic"))
	if err != nil {
		return err
	}
	a.router = router

	return nil
}

func (a *App) syncToRouter(
	ctx context.Context,
	routes []keenetic.IPRoute,
) error {
	currentRoutes, err := a.router.LoadIPRoutes()
	if err != nil {
		return fmt.Errorf("loading current routes failed: %w", err)
	}

	newRoutes := a.filterNewRoutes(currentRoutes, routes)
	newRoutesCount := len(newRoutes)

	outdatedRoutes := a.filterOutdatedRoutes(currentRoutes, routes)
	outdatedRoutesCount := len(outdatedRoutes)

	if newRoutesCount+outdatedRoutesCount > 0 {
		a.logger.Info(
			"syncing routes to router",
			slog.Int("new", newRoutesCount),
			slog.Int("outdated", outdatedRoutesCount),
		)
	} else {
		a.logger.Info("nothing to syncing to router")
	}

	if newRoutesCount > 0 {
		err := a.router.AddIPRoutes(ctx, newRoutes)
		if err != nil {
			return fmt.Errorf("adding new routes failed: %w", err)
		}
	}
	if outdatedRoutesCount > 0 {
		err := a.router.RemoveIPRoutes(ctx, outdatedRoutes)
		if err != nil {
			return fmt.Errorf("removing outdated routes failed: %w", err)
		}
	}

	return nil
}

func (a *App) resolveRoutes(ctx context.Context) ([]keenetic.IPRoute, time.Time, error) {
	var (
		unresolved   []*resolveEntry
		nextExpireAt = time.Now().Add(defaultMinTTL)
	)

	if a.resolveQueue == nil {
		unresolved = newResolveEntries(a)
		a.resolveQueue = &resolveQueue{}
	} else {
		unresolved = a.resolveQueue.popExpiredRoutes()
		if a.resolveQueue.Len() > 0 {
			nextExpireAt = (*a.resolveQueue)[0].expireAt
		}
	}

	routes := make([]keenetic.IPRoute, 0)
	for _, entry := range *a.resolveQueue {
		routes = append(routes, entry.routes...)
	}

	for _, entry := range unresolved {
		now := time.Now()
		if now.After(entry.expireAt) {
			a.resolveRouteEntry(ctx, entry)
		}
		heap.Push(a.resolveQueue, entry)

		if nextExpireAt.After(entry.expireAt) {
			nextExpireAt = entry.expireAt
		}

		slices.Grow(routes, len(entry.routes))
		routes = append(routes, entry.routes...)
	}

	return routes, nextExpireAt, nil
}

func (a *App) resolveRouteEntry(
	ctx context.Context,
	entry *resolveEntry,
) time.Duration {
	addresses, err := a.resolver.Resolve(
		ctx,
		entry.target,
		entry.resolver,
		entry.filters,
	)

	if err != nil {
		if !errors.Is(err, context.Canceled) {
			a.logger.Error("could not resolve addresses", slog.Any("error", err))
		}
		return defaultMinTTL
	}

	var (
		minTTL = defaultMinTTL
		now    = time.Now()
		routes = make([]keenetic.IPRoute, 0, len(addresses))
	)

	for _, address := range addresses {
		if address.HasTTL() {
			minTTL = min(minTTL, address.TTL)
		}
		routes = append(routes, keenetic.IPRoute{
			Destination: address.Addr,
			Interface:   entry.interfaceName,
			Gateway:     entry.gateway,
			Auto:        entry.auto,
			Description: address.Description,
		})
	}

	entry.resolved(routes, now.Add(minTTL))

	return minTTL
}

func (a *App) filterNewRoutes(currentRoutes, routes []keenetic.IPRoute) []keenetic.IPRoute {
	return slices.DeleteFunc(
		slices.Clone(routes),
		func(newRoute keenetic.IPRoute) bool {
			return slices.ContainsFunc(currentRoutes, newRoute.Equals)
		},
	)
}

func (a *App) filterOutdatedRoutes(currentRoutes, routes []keenetic.IPRoute) []keenetic.IPRoute {
	cleanupInterfaceNames := make([]string, 0, len(a.conf.Interfaces))
	for _, interfaceCfg := range a.conf.Interfaces {
		if interfaceCfg.Cleanup {
			cleanupInterfaceNames = append(cleanupInterfaceNames, interfaceCfg.Name)
		}
	}

	return slices.DeleteFunc(
		slices.Clone(currentRoutes),
		func(route keenetic.IPRoute) bool {
			return route.IsProtected() ||
				!slices.Contains(cleanupInterfaceNames, route.Interface) ||
				slices.ContainsFunc(routes, route.Equals)
		},
	)
}
