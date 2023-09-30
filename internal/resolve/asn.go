package resolve

import (
	"context"
	"fmt"
	"strconv"

	"github.com/idr0id/keenctl/internal/asn"
	"github.com/idr0id/keenctl/internal/network"
)

func resolveAsn(ctx context.Context, s string) ([]network.Addr, error) {
	number, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("asn `%s` must be number: %w", s, err)
	}

	prefixes, err := asn.GetAnnouncedPrefixes(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve asn `%d`: %w", number, err)
	}

	result := make([]network.Addr, len(prefixes))
	for i, p := range prefixes {
		result[i], err = network.ParseCIDR(p)
		if err != nil {
			return nil, fmt.Errorf("unable to parse prefix `%s` of asn `%d`: %w", p, number, err)
		}
	}

	return result, nil
}
