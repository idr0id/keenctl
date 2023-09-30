// Package resolve provides mechanisms for resolving network addresses.
package resolve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/idr0id/keenctl/internal/network"
)

// Address represents a network address with an associated description.
type Address struct {
	Addr        network.Addr
	Description string
}

type (
	addressResolver func(context.Context, string) ([]network.Addr, error)
	addressFilter   func(network.Addr) bool
)

var (
	ErrResolverNotFound = errors.New("address resolver not found")
	ErrFilterNotFound   = errors.New("address filter not found")
)

var (
	resolvers = map[string]addressResolver{
		"dns":  resolveDNS,
		"asn":  resolveAsn,
		"addr": resolveAddr,
	}
	filters = map[string]addressFilter{
		"ipv4":        filterIPv4,
		"ipv6":        filterIPv6,
		"private":     filterPrivate,
		"loopback":    filterLoopback,
		"unspecified": filterUnspecified,
	}
)

// Addresses parse configuration and resolve routes addresses.
func Addresses(
	ctx context.Context,
	logger *slog.Logger,
	target string,
	resolverName string,
	filterNames []string,
) ([]Address, error) {
	if resolverName == "" {
		resolverName = "addr"
	}

	resolver, ok := resolvers[resolverName]
	if !ok {
		return nil, fmt.Errorf("address resolver: %s: %w", resolverName, ErrResolverNotFound)
	}

	resolved, err := resolver(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("address resolver: %s: %w", resolverName, err)
	}

	addressFilters, err := getFilters(filterNames)
	if err != nil {
		return nil, fmt.Errorf("address filter: %w", err)
	}
	filtered := filter(resolved, addressFilters)

	logger.Debug(
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
		}
	}

	return addresses, nil
}

func filter(addresses []network.Addr, filters []addressFilter) []network.Addr {
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
	return result
}

func getFilters(filterNames []string) ([]addressFilter, error) {
	result := make([]addressFilter, 0, len(filterNames))

	for _, filterName := range filterNames {
		f, ok := filters[filterName]
		if !ok {
			return nil, fmt.Errorf("%s: %w", filterName, ErrFilterNotFound)
		}
		result = append(result, f)
	}

	return result, nil
}

func formatDescription(target, resolver string) string {
	if resolver != "" {
		return fmt.Sprintf("%s(%s)", resolver, target)
	}
	return target
}
