package resolve

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveDNS(t *testing.T) {
	configs := []DnsConfig{
		{Nameservers: nil},
	}

	for _, conf := range configs {
		resolver := newDNSResolver(DnsConfig{})
		addresses, err := resolver.resolve(context.Background(), "example.com.")

		require.NoError(t, err, fmt.Sprintf("configuration: %v", conf))
		require.GreaterOrEqual(
			t, len(addresses), 1,
			fmt.Sprintf("Expected at least one address; configuration: %v", conf),
		)
	}
}
