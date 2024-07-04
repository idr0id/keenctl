package cmd

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	application "github.com/idr0id/keenctl/internal/app"
	"github.com/spf13/cobra"
)

func newServeCommand() *cobra.Command {
	var (
		configPath string
		dryRun     bool
		verbose    bool
		quiet      bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Runs in serve mode",
		RunE: func(_ *cobra.Command, _ []string) error {
			logger := setupLogger(verbose, quiet)

			logger.Info("parsing configuration")
			conf, err := application.ParseConfig(configPath, dryRun)
			if err != nil {
				logger.Error("configuration error", slog.Any("error", err))
				return errSilent
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			registerSignalHandler(logger, cancel)

			app := application.New(conf, logger)
			if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("application error", slog.Any("error", err))
				return errSilent
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "./keenctl.toml", "Path to the configuration file [default: ./keenctl.toml]")
	cmd.Flags().BoolVar(&dryRun, "dryRun", false, "Executes a dry run without changing routes")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Print debug information on stderr")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Silent mode")

	return cmd
}

func registerSignalHandler(logger *slog.Logger, cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	go func() {
		signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
		<-signals

		signal.Stop(signals)
		logger.Info("shutting down")
		cancel()
	}()
}
