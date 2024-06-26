package app

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/idr0id/keenctl/internal/keenetic"
	"github.com/idr0id/keenctl/internal/resolve"
)

type App struct {
	conf          Config
	logger        *slog.Logger
	router        *keenetic.Router
	currentRoutes []keenetic.IPRoute

	wg     sync.WaitGroup
	doneCh chan struct{}
}

func New(conf Config, logger *slog.Logger) *App {
	return &App{
		conf:   conf,
		logger: logger,
		doneCh: make(chan struct{}, 1),
	}
}

func (a *App) Run() error {
	routesCh := make(chan []keenetic.IPRoute)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		t := time.NewTimer(0)

		for {
			select {
			case <-t.C:
				a.logger.Info("resolving addresses for routes")
				routes, err := a.resolveRoutes(context.Background())
				if err != nil {
					a.logger.Error("failed to resolve addresses for routes", slog.Any("error", err))
					break
				}
				routesCh <- routes
				t.Reset(time.Minute)

			case <-a.doneCh:
				close(routesCh)
				return
			}
		}

	}()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		if err := a.routerRun(routesCh); err != nil {
			a.logger.Error("unable to start router", "error", err)
		}
	}()

	a.wg.Wait()

	return nil
}

func (a *App) Shutdown() {
	close(a.doneCh)
	a.wg.Wait()
}

func (a *App) routerRun(routesCh chan []keenetic.IPRoute) error {
	a.logger.Info("connecting to router")
	router, err := keenetic.Connect(a.conf.SSH, a.logger.WithGroup("keenetic"))
	if err != nil {
		return err
	}
	a.router = router

	currentRoutes, err := router.LoadIPRoutes()
	if err != nil {
		return err
	}
	a.currentRoutes = currentRoutes

	for routes := range routesCh {
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
	}

	return nil
}

func (a *App) resolveRoutes(ctx context.Context) ([]keenetic.IPRoute, error) {
	var routes []keenetic.IPRoute

	for _, interfaceConf := range a.conf.Interfaces {
		for _, routeConf := range interfaceConf.Routes {
			addresses, err := resolve.Addresses(
				ctx,
				a.logger,
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
