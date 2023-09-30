package keenetic

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/idr0id/keenctl/internal/network"
)

// IPRoute represents a network route in the routing table.
// It includes the necessary information to direct traffic to a specific network or host.
type IPRoute struct {
	Destination network.Addr
	Interface   string
	Gateway     string
	Flags       string
	Description string
	Metrics     uint32
	Auto        bool
}

// ErrParseRoutes is the error when routes has invalid format.
var ErrParseRoutes = errors.New("invalid format of routes")

// Equals reports whether r equals to o.
func (r *IPRoute) Equals(o IPRoute) bool {
	return r.Interface == o.Interface && r.Destination.Contains(o.Destination)
}

// IsProtected reports whether destination address is protected from modification.
func (r *IPRoute) IsProtected() bool {
	return r.Destination.IP.IsPrivate() ||
		r.Destination.IP.IsLoopback() ||
		r.Destination.IP.IsUnspecified()
}

func parseIPRoutes(stdout string) ([]IPRoute, error) {
	if stdout == "" {
		return nil, nil
	}

	lines := strings.Split(strings.Trim(stdout, "\n"), "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("%w: %s", ErrParseRoutes, stdout)
	}

	routes := make([]IPRoute, 0, len(lines)-3)
	for _, line := range lines[3:] {
		if route, err := parseLine(line); err == nil {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

func parseLine(line string) (IPRoute, error) {
	columns := make([]string, 5)

	for current, prev, column := 0, 0, 0; current < len(line); current++ {
		if line[current] != ' ' {
			continue
		}

		if prev != current {
			columns[column] = line[prev:current]
			column++
		}

		prev = current + 1
	}

	metrics, _ := strconv.ParseUint(columns[4], 10, 32)

	ip, err := network.ParseCIDR(columns[0])

	return IPRoute{
		Destination: ip,
		Gateway:     columns[1],
		Interface:   columns[2],
		Flags:       columns[3],
		Metrics:     uint32(metrics),
	}, err
}
