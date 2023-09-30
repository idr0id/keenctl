// Package network deals with network routing configurations.
package network

import (
	"net"
)

// Addr represents and encapsulate `net.IPNet`.
type Addr struct {
	*net.IPNet
}

// ParseCIDR parses s as a CIDR notation.
func ParseCIDR(s string) (Addr, error) {
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		return Addr{}, err
	}
	return Addr{IPNet: ipNet}, nil
}

// ParseIP parses s as IP notation.
func ParseIP(s string) Addr {
	return IP(net.ParseIP(s))
}

// IP converts ip to Addr.
func IP(ip net.IP) Addr {
	bits := 32
	if len(ip) == net.IPv6len {
		bits = 128
	}

	mask := net.CIDRMask(bits, bits)
	return Addr{IPNet: &net.IPNet{IP: ip, Mask: mask}}
}

// IsIPv4 reports whether network address is v4.
func (addr Addr) IsIPv4() bool {
	return len(addr.IP) == net.IPv4len
}

// IsIPv6 reports whether network address is v6.
func (addr Addr) IsIPv6() bool {
	return len(addr.IP) == net.IPv6len
}

// Contains reports whether the network includes n.
func (addr Addr) Contains(n Addr) bool {
	return addr.IPNet.Contains(n.IP)
}

// String returns the CIDR notation of addr.
func (addr Addr) String() string {
	if addr.IPNet == nil {
		return "<nil>"
	}
	return addr.IPNet.String()
}
