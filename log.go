package main

import "github.com/rs/zerolog"

func parseVerbosity(args map[string]interface{}) {
	level := args["--verbose"].(int)

	switch true {
	case level > 1:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case level == 1:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
