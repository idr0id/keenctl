package app

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/idr0id/keenctl/internal/keenetic"
	"github.com/idr0id/keenctl/internal/resolve"
	"golang.org/x/sync/errgroup"
)

const maxRetryDelay = 5 * time.Second

type App struct {
	conf          Config
	logger        *slog.Logger
	router        *keenetic.Router
	resolver      *resolve.Resolver
	currentRoutes []keenetic.IPRoute

	eg     errgroup.Group
	doneCh chan struct{}
}

func New(conf Config, logger *slog.Logger) *App {
	return &App{
		conf:     conf,
		logger:   logger,
		resolver: resolve.New(logger),
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
				a.logger.Error("syncing to router failed: %s", slog.Any("error", err))
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
				routes, err := a.resolveRoutes(ctx)
				if err != nil {
					a.logger.Error("resolving an addresses for routes failed", slog.Any("error", err))
					continue
				}
				routesCh <- routes
				t.Reset(time.Minute)

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
				a.logger.Info("sync: shutting down")
				return nil

			case routes := <-routesCh:
				if err := a.syncToRouter(routes); err != nil {
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

	currentRoutes, err := a.router.LoadIPRoutes()
	if err != nil {
		return err
	}
	a.currentRoutes = currentRoutes
	return nil
}

func (a *App) syncToRouter(routes []keenetic.IPRoute) error {
	removedCount, err := a.removeRoutes(routes)
	if err != nil {
		return err
	}

	addedCount, err := a.addRoutes(routes)
	if err != nil {
		return err
	}

	if addedCount+removedCount > 0 {
		a.logger.Info(
			"routes successfully updated",
			slog.Int("added", addedCount),
			slog.Int("removed", removedCount),
		)
	} else {
		a.logger.Info("nothing to update")
	}
	return nil
}

func (a *App) resolveRoutes(ctx context.Context) ([]keenetic.IPRoute, error) {
	var routes []keenetic.IPRoute

	for _, interfaceConf := range a.conf.Interfaces {
		for _, routeConf := range interfaceConf.Routes {
			addresses, err := a.resolver.Resolve(
				ctx,
				routeConf.Target,
				routeConf.Resolver,
				routeConf.GetFilters(interfaceConf.Defaults),
			)
			if err != nil {
				a.logger.Warn("could not resolve addresses", slog.Any("error", err))
				continue
			}

			routes = slices.Grow(routes, len(addresses))

			for _, address := range addresses {
				routes = append(routes, keenetic.IPRoute{
					Destination: address.Addr,
					Interface:   interfaceConf.Name,
					Gateway:     routeConf.GetGateway(interfaceConf.Defaults),
					Auto:        routeConf.GetAuto(interfaceConf.Defaults),
					Description: address.Description,
				})
			}
		}
	}

	return routes, nil
}

func (a *App) removeRoutes(routes []keenetic.IPRoute) (int, error) {
	cleanupInterfaceNames := make([]string, 0, len(a.conf.Interfaces))
	for _, interfaceCfg := range a.conf.Interfaces {
		if interfaceCfg.Cleanup {
			cleanupInterfaceNames = append(cleanupInterfaceNames, interfaceCfg.Name)
		}
	}

	outdatedRoutes := slices.DeleteFunc(
		slices.Clone(a.currentRoutes),
		func(route keenetic.IPRoute) bool {
			return route.IsProtected() ||
				!slices.Contains(cleanupInterfaceNames, route.Interface) ||
				slices.ContainsFunc(routes, route.Equals)
		},
	)

	if len(outdatedRoutes) == 0 {
		return 0, nil
	}

	return len(outdatedRoutes), a.router.RemoveIPRoutes(outdatedRoutes)
}

func (a *App) addRoutes(newRoutes []keenetic.IPRoute) (int, error) {
	// dont add exists routes twice
	newRoutes = slices.DeleteFunc(
		slices.Clone(newRoutes),
		func(newRoute keenetic.IPRoute) bool {
			return slices.ContainsFunc(a.currentRoutes, newRoute.Equals)
		},
	)

	if len(newRoutes) == 0 {
		return 0, nil
	}

	return len(newRoutes), a.router.AddIPRoutes(newRoutes)
}
