package resolve

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/idr0id/keenctl/internal/network"
	"github.com/miekg/dns"
)

type dnsResolver struct {
	client *dns.Client
	conf   DnsConfig
}

func newDNSResolver(conf DnsConfig) *dnsResolver {
	return &dnsResolver{
		client: &dns.Client{},
		conf:   conf,
	}
}

func (r *dnsResolver) resolve(ctx context.Context, host string) ([]network.Addr, error) {
	answerA, err := r.sendQuestion(ctx, host, dns.TypeA)
	if err != nil {
		return nil, err
	}

	answerAAAA, err := r.sendQuestion(ctx, host, dns.TypeAAAA)
	if err != nil {
		return nil, err
	}

	resolved := make([]network.Addr, 0, len(answerA)+len(answerAAAA))
	for _, answer := range append(answerA, answerAAAA...) {
		switch record := answer.(type) {
		case *dns.A:
			resolved = append(resolved, network.IP(record.A))

		case *dns.AAAA:
			resolved = append(resolved, network.IP(record.AAAA))

		case *dns.CNAME:
			// do nothing

		default:
			return nil, fmt.Errorf("dns resolver: unexpected answer type: %T", record)
		}
	}

	return resolved, nil
}

func (r *dnsResolver) getNameservers() ([]string, error) {
	if len(r.conf.Nameservers) > 0 {
		return formatNameservers(r.conf.Nameservers), nil
	}

	resolvConf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, fmt.Errorf("unable to read /etc/resolv.conf: %w", err)
	}

	return formatNameservers(resolvConf.Servers), nil
}

func (r *dnsResolver) sendQuestion(ctx context.Context, host string, dnsType uint16) ([]dns.RR, error) {
	nameservers, err := r.getNameservers()
	if err != nil {
		return nil, err
	}

	msg := &dns.Msg{}
	msg.Id = dns.Id()
	msg.SetQuestion(fqdn(host), dnsType)

	var serverErr error
	for _, serverAddress := range nameservers {
		response, _, err := r.client.ExchangeContext(ctx, msg, serverAddress)
		if err != nil {
			serverErr = err
			continue
		}
		if response.Rcode == dns.RcodeSuccess {
			return response.Answer, nil
		}
		serverErr = fmt.Errorf("dns: resolution failed: %s", response.String())
	}

	return nil, serverErr
}

func fqdn(host string) string {
	if strings.HasSuffix(host, ".") {
		return host
	}
	return host + "."
}

func formatNameservers(hosts []string) []string {
	addresses := make([]string, len(hosts))
	for i, host := range hosts {
		addresses[i] = net.JoinHostPort(host, "53")
	}
	return addresses
}
