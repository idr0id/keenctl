package main

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net"
	"strconv"
	"strings"
	"time"
)

type IpRoute struct {
	Destination   IpRouteAddress
	Gateway       string
	InterfaceName string
	Flags         string
	Metrics       uint32
}

type IpRouteList []IpRoute

type IpRouteProvisionList []IpRouteProvisionAddress

type IpRouteAddress string

type IpRouteProvisionAddress struct {
	Destination IpRouteAddress
	Comment     string
}

type IpRouteAddressResolver func(input string) ([]string, error)

func (addr IpRouteAddress) Equals(other IpRouteAddress) bool {
	addrNoSuffix, _ := strings.CutSuffix(addr.String(), "/32")
	otherNoSuffix, _ := strings.CutSuffix(addr.String(), "/32")
	return addrNoSuffix == otherNoSuffix
}

func (addr IpRouteAddress) String() string {
	return string(addr)
}

func parseIpRoutes(stdout string) IpRouteList {
	if stdout == "" {
		return nil
	}

	lines := strings.Split(strings.Trim(stdout, "\n"), "\n")
	if len(lines) < 3 {
		log.Fatal().Bytes("stdout", []byte(stdout)).Msg("invalid format of routes table")
	}

	list := make(IpRouteList, len(lines)-3)
	for i, line := range lines[3:] {
		list[i] = parseIpRoute(line)
	}
	return list
}

func parseIpRoute(line string) IpRoute {
	columns := make([]string, 5)

	for i, prev, column := 0, 0, 0; i < len(line); i++ {
		if line[i] != ' ' {
			continue
		}
		if prev != i {
			columns[column] = line[prev:i]
			column++
		}
		prev = i + 1
	}

	metrics, _ := strconv.ParseUint(columns[4], 10, 32)

	route := IpRoute{
		Destination:   IpRouteAddress(columns[0]),
		Gateway:       columns[1],
		InterfaceName: columns[2],
		Flags:         columns[3],
		Metrics:       uint32(metrics),
	}

	log.Trace().
		Str("line", line).
		Interface("route", route).
		Msg("route parsed")

	return route
}

func (l IpRouteList) filterIpRoutes(predicate func(route IpRoute) bool) IpRouteList {
	out := make(IpRouteList, 0)
	for _, route := range l {
		if predicate(route) {
			out = append(out, route)
		}
	}
	return out
}

func makeIpRouteAddressResolver(
	resolverName string,
	l zerolog.Logger,
) IpRouteAddressResolver {
	switch resolverName {
	case "lookup-ip":
		return lookupIpAddressResolver

	case "lookup-asn":
		return lookupAsnAddressResolver

	case "filter-ipv6":
		return filterIpv6AddressResolver

	default:
		l.Fatal().Str("resolver", resolverName).Msg("unknown resolver")
	}
	return nil
}

func lookupIpAddressResolver(input string) ([]string, error) {
	ips, err := net.LookupIP(input)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(ips))
	for i, ip := range ips {
		out[i] = ip.String()
	}
	return out, nil
}

func lookupAsnAddressResolver(input string) ([]string, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	return GetAsnAnnouncedPrefixes(input, ctx), nil
}

func filterIpv6AddressResolver(input string) ([]string, error) {
	if strings.Count(input, ":") >= 2 {
		return nil, nil
	}
	return []string{input}, nil
}
