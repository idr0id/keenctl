package resolve

import "github.com/idr0id/keenctl/internal/network"

func filterIPv4(addr network.Addr) bool {
	return addr.IsIPv4()
}

func filterIPv6(addr network.Addr) bool {
	return addr.IsIPv6()
}

func filterPrivate(addr network.Addr) bool {
	return addr.IP.IsPrivate()
}

func filterLoopback(addr network.Addr) bool {
	return addr.IP.IsLoopback()
}

func filterUnspecified(addr network.Addr) bool {
	return addr.IP.IsUnspecified()
}
