package resolve

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveDNS(t *testing.T) {
	addresses, err := resolveDNS(context.Background(), "example.com.")

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(addresses), 1, "Expected at least one address")
}
