package resolve

import (
	"context"
	"net"

	"github.com/idr0id/keenctl/internal/network"
)

func resolveDNS(_ context.Context, s string) ([]network.Addr, error) {
	ips, err := net.LookupIP(s)
	if err != nil {
		return nil, err
	}

	result := make([]network.Addr, len(ips))
	for i, ip := range ips {
		result[i] = network.IP(ip)
	}

	return result, nil
}
