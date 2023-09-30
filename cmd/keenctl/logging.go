package main

import (
	"log/slog"
	"os"
)

func setupLogger(args map[string]any) *slog.Logger {
	var (
		quiet, _   = args["--quiet"].(bool)
		verbose, _ = args["--verbose"].(int)
	)

	var level slog.Level
	switch {
	case quiet:
		level = slog.LevelWarn
	case verbose == 1:
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}
