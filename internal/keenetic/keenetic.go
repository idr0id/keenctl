// Package keenetic provides functionalities to interact with Keenetic routers.
package keenetic

import (
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"
)

// Router manages SSH connections and executing commands to a network router.
type Router struct {
	connPool chan *sshClient
	logger   *slog.Logger
	dryRun   bool
}

// ErrMaxParallelCommands is the error when MaxParallelCommands is <= 0.
var ErrMaxParallelCommands = errors.New("MaxParallelCommands must be greater than zero")

// Connect initializes a new connection to Router using the provided configuration.
func Connect(conf ConnConfig, logger *slog.Logger) (*Router, error) {
	if conf.MaxParallelCommands == 0 {
		return nil, ErrMaxParallelCommands
	}

	router := &Router{
		connPool: make(chan *sshClient, conf.MaxParallelCommands),
		logger:   logger,
		dryRun:   conf.DryRun,
	}

	var g errgroup.Group
	for range conf.MaxParallelCommands {
		g.Go(func() error {
			sshClient, err := newSSHClient(conf)
			if err == nil {
				router.connPool <- sshClient
			}

			return err
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return router, nil
}

// LoadIPRoutes retrieves the current IP routing table from the router.
func (r *Router) LoadIPRoutes() ([]IPRoute, error) {
	output, err := r.exec("show ip route")
	if err != nil {
		return nil, err
	}

	return parseIPRoutes(output)
}

// AddIPRoute adds a new IP route to the router's routing table.
func (r *Router) AddIPRoute(route IPRoute) error {
	if r.dryRun {
		return nil
	}

	// ip route 10.0.0.1 Wireguard0 auto !example
	auto := ""
	if route.Auto {
		auto = "auto"
	}

	cmd := fmt.Sprintf(
		"ip route %s %s %s !%s",
		route.Destination,
		route.Interface,
		auto,
		route.Description,
	)
	_, err := r.exec(cmd)

	return err
}

func (r *Router) AddIPRoutes(routes []IPRoute) error {
	var g = errgroup.Group{}

	for _, route := range routes {
		route := route
		g.Go(func() error {
			return r.AddIPRoute(route)
		})
	}

	return g.Wait()
}

// RemoveIPRoute removes an IP route from the router's routing table.
func (r *Router) RemoveIPRoute(rout IPRoute) error {
	if r.dryRun {
		return nil
	}

	cmd := fmt.Sprintf(
		"no ip route %s %s",
		rout.Destination,
		rout.Interface,
	)
	_, err := r.exec(cmd)

	return err
}

func (r *Router) RemoveIPRoutes(routes []IPRoute) error {
	var g = errgroup.Group{}

	for _, route := range routes {
		route := route
		g.Go(func() error {
			return r.RemoveIPRoute(route)
		})
	}

	return g.Wait()
}

func (r *Router) exec(cmd string) (string, error) {
	sshClient := <-r.connPool
	defer func() {
		r.connPool <- sshClient
	}()

	if r.logger != nil {
		r.logger.Debug("execute command", slog.Any("cmd", cmd))
	}

	out, err := sshClient.exec(cmd)
	if err != nil {
		return "", fmt.Errorf("%s: %w", cmd, err)
	}

	return string(out), nil
}
