package app

import (
	"context"
	"log/slog"
	"slices"

	"github.com/idr0id/keenctl/internal/keenetic"
	"github.com/idr0id/keenctl/internal/resolve"
)

type App struct {
	conf          Config
	logger        *slog.Logger
	router        *keenetic.Router
	currentRoutes []keenetic.IPRoute
}

func New(conf Config, logger *slog.Logger) *App {
	return &App{
		conf:   conf,
		logger: logger,
	}
}

func (a *App) Run() error {
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

	a.logger.Info("starting to resolve addresses for routes")
	routes, err := a.resolveRoutes()
	if err != nil {
		return err
	}

	if err := a.removeOutdatedRoutes(routes); err != nil {
		return err
	}

	if err := a.addNewRoutes(routes); err != nil {
		return err
	}

	return nil
}

func (a *App) resolveRoutes() ([]keenetic.IPRoute, error) {
	var routes []keenetic.IPRoute

	for _, interfaceConf := range a.conf.Interfaces {
		for _, routeConf := range interfaceConf.Routes {
			addresses, err := resolve.Addresses(
				context.Background(),
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

func (a *App) removeOutdatedRoutes(routes []keenetic.IPRoute) error {
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
		return nil
	}

	a.logger.Info(
		"cleanup outdated routes",
		slog.Int("count", len(outdatedRoutes)),
	)

	return a.router.RemoveIPRoutes(outdatedRoutes)
}

func (a *App) addNewRoutes(newRoutes []keenetic.IPRoute) error {
	// dont add exists routes twice
	newRoutes = slices.DeleteFunc(
		slices.Clone(newRoutes),
		func(newRoute keenetic.IPRoute) bool {
			return slices.ContainsFunc(a.currentRoutes, newRoute.Equals)
		},
	)

	if len(newRoutes) == 0 {
		a.logger.Info("no new routes to add")
		return nil
	}

	a.logger.Info("add new routes", slog.Int("count", len(newRoutes)))

	return a.router.AddIPRoutes(newRoutes)
}
