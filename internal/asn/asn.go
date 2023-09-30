// Package asn provides functionalities for working with Autonomous Systems (AS).
package asn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// AnnouncedPrefixes lists the list of IP ranges owned by the AS.
type AnnouncedPrefixes []string

type responseData struct {
	Data struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	} `json:"data"`
}

const baseURL = "https://stat.ripe.net/data/announced-prefixes/data.json?resource=AS%d&sourceapp=keenctl"

// ASN errors.
var (
	ErrUnmarshal      = errors.New("error unmarshal response body")
	ErrHTTPStatusCode = errors.New("http returned error status code")
)

// GetAnnouncedPrefixes returns the list of IP ranges owned by the AS.
func GetAnnouncedPrefixes(ctx context.Context, number int) (AnnouncedPrefixes, error) {
	url := fmt.Sprintf(baseURL, number)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)

		return nil, fmt.Errorf(
			"%w: code %d: body %s",
			ErrHTTPStatusCode,
			response.StatusCode,
			string(body),
		)
	}

	var data responseData

	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnmarshal, err)
	}

	result := make(AnnouncedPrefixes, len(data.Data.Prefixes))
	for i, prefix := range data.Data.Prefixes {
		result[i] = prefix.Prefix
	}

	return result, nil
}
