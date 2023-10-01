package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
)

type AsnAnnouncedPrefixes []string

const asnAnnouncedPrefixesUrl = "https://stat.ripe.net/data/announced-prefixes/" +
	"data.json?resource=AS%s&sourceapp=routek"

func GetAsnAnnouncedPrefixes(asn string, ctx context.Context) AsnAnnouncedPrefixes {
	url := fmt.Sprintf(
		asnAnnouncedPrefixesUrl,
		asn,
	)

	l := log.With().Str("asn", asn).Str("url", url).Logger()

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		l.Fatal().Err(err).Msg("http request failed")
	}

	response, err := http.DefaultClient.Do(request.WithContext(ctx))
	defer func() { _ = response.Body.Close() }()
	if err != nil {
		l.Fatal().Err(err).Msg("http request failed")
	}

	if response.StatusCode != 200 {
		body, _ := io.ReadAll(response.Body)
		l.Fatal().
			Str("response_status", response.Status).
			Int("response_status_code", response.StatusCode).
			Str("response_body", string(body)).
			Msg("http request failed")
	}

	var data struct {
		Data struct {
			Prefixes []struct {
				Prefix string `json:"prefix"`
			} `json:"prefixes"`
		} `json:"data"`
	}

	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		body, _ := io.ReadAll(response.Body)
		l.Fatal().
			Err(err).
			Str("body", string(body)).
			Msg("can't unmarshal http response")
	}

	l.Debug().
		Int("count", len(data.Data.Prefixes)).
		Msg("found asn announced prefixes")

	out := make(AsnAnnouncedPrefixes, len(data.Data.Prefixes))
	for i, prefix := range data.Data.Prefixes {
		out[i] = prefix.Prefix
	}
	return out
}
