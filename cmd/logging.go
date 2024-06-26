package cmd

import (
	"log/slog"
	"os"
)

func setupLogger(verbose, quiet bool) *slog.Logger {
	var level slog.Level

	switch {
	case quiet:
		level = slog.LevelError
	case verbose:
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}
