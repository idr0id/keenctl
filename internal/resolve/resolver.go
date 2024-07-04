// Package resolve provides mechanisms for resolving network addresses.
package resolve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/idr0id/keenctl/internal/network"
)

// Address represents a network address with an associated description.
type Address struct {
	Addr        network.Addr
	Description string
	TTL         time.Duration
}

// HasTTL checks if the Address has a Time-To-Live (TTL) value greater than zero.
func (a Address) HasTTL() bool {
	return a.TTL > 0
}

type (
	addressResolver func(context.Context, string) ([]network.Addr, error)
	addressFilter   func(network.Addr) bool
)

var (
	ErrResolverNotFound = errors.New("address resolver not found")
	ErrFilterNotFound   = errors.New("address filter not found")
)

type Resolver struct {
	logger    *slog.Logger
	resolvers map[string]addressResolver
	filters   map[string]addressFilter
}

func New(conf ResolverConfig, logger *slog.Logger) *Resolver {
	return &Resolver{
		logger: logger,
		resolvers: map[string]addressResolver{
			"dns":  newDNSResolver(conf.DNS).resolve,
			"asn":  resolveAsn,
			"addr": resolveAddr,
		},
		filters: map[string]addressFilter{
			"ipv4":        filterIPv4,
			"ipv6":        filterIPv6,
			"private":     filterPrivate,
			"loopback":    filterLoopback,
			"unspecified": filterUnspecified,
		},
	}
}

// Resolve parse configuration and resolve routes addresses.
func (r *Resolver) Resolve(
	ctx context.Context,
	target string,
	resolverName string,
	filterNames []string,
) ([]Address, error) {
	if resolverName == "" {
		resolverName = "addr"
	}

	resolver, ok := r.resolvers[resolverName]
	if !ok {
		return nil, fmt.Errorf("address resolver: %s: %w", resolverName, ErrResolverNotFound)
	}

	resolved, err := resolver(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("address resolver: %s: %w", resolverName, err)
	}

	filtered, err := r.filter(resolved, filterNames)
	if err != nil {
		return nil, fmt.Errorf("address filter: %w", err)
	}

	r.logger.Debug(
		"resolver: addresses resolved successfully",
		slog.String("target", target),
		slog.String("resolver", resolverName),
		slog.Any("filters", filterNames),
		slog.Int("resolved", len(resolved)),
		slog.Int("filtered", len(filtered)),
	)

	if len(filtered) == 0 {
		return nil, nil
	}

	var (
		addresses   = make([]Address, len(filtered))
		description = formatDescription(target, resolverName)
	)

	for i, addr := range filtered {
		addresses[i] = Address{
			Addr:        addr,
			Description: description,
			TTL:         addr.TTL,
		}
	}

	return addresses, nil
}

func (r *Resolver) filter(
	addresses []network.Addr,
	filterNames []string,
) ([]network.Addr, error) {
	filters := make([]addressFilter, 0, len(filterNames))
	for _, filterName := range filterNames {
		filter, ok := r.filters[filterName]
		if !ok {
			return nil, fmt.Errorf("%s: %w", filterName, ErrFilterNotFound)
		}
		filters = append(filters, filter)
	}

	result := make([]network.Addr, 0, len(addresses))
	for _, address := range addresses {
		ok := true
		for _, filter := range filters {
			ok = ok && !filter(address)
		}
		if ok {
			result = append(result, address)
		}
	}
	return result, nil
}

func formatDescription(target, resolver string) string {
	if resolver != "" {
		return fmt.Sprintf("%s(%s)", resolver, target)
	}
	return target
}
