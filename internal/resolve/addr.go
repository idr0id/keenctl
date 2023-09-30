package resolve

import (
	"context"
	"fmt"

	"github.com/idr0id/keenctl/internal/network"
)

func resolveAddr(_ context.Context, s string) ([]network.Addr, error) {
	addr, err := network.ParseCIDR(s)
	if err != nil {
		addr = network.ParseIP(s)
		if addr.IP == nil {
			return nil, fmt.Errorf("invalid address: %s", s)
		}
	}

	return []network.Addr{addr}, nil
}
