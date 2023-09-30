package asn_test

import (
	"context"
	"testing"

	"github.com/idr0id/keenctl/internal/asn"

	"github.com/stretchr/testify/require"
)

func TestGetAnnouncedPrefixes(t *testing.T) {
	prefixes, err := asn.GetAnnouncedPrefixes(context.Background(), 1)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(prefixes), 1)
}

func TestGetAnnouncedPrefixesInvalidNumber(t *testing.T) {
	_, err := asn.GetAnnouncedPrefixes(context.Background(), -1)

	require.Error(t, err)
}
