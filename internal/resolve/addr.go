package resolve

import (
	"context"
	"errors"
	"fmt"

	"github.com/idr0id/keenctl/internal/network"
)

// ErrInvalidAddress is a common error for address resolver.
var ErrInvalidAddress = errors.New("invalid address")

func resolveAddr(_ context.Context, s string) ([]network.Addr, error) {
	addr, err := network.ParseCIDR(s)
	if err == nil {
		return []network.Addr{addr}, nil
	}

	addr = network.ParseIP(s)
	if addr.IP == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAddress, s)
	}

	return []network.Addr{addr}, nil
}
