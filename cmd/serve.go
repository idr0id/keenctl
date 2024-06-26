package cmd

import (
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
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := setupLogger(verbose, quiet)

			logger.Info("parsing configuration")
			conf, err := application.ParseConfig(configPath, dryRun)
			if err != nil {
				logger.Error("configuration error", slog.Any("error", err))
				return silentErr
			}

			app := application.New(conf, logger)
			appExitCh := make(chan error)

			go func() {
				appExitCh <- app.Run()
			}()

			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

			for {
				select {
				case <-signals:
					logger.Info("shutting down")
					app.Shutdown()
					return nil

				case err := <-appExitCh:
					if err != nil {
						logger.Error("application error", slog.Any("error", err))
						return silentErr
					}
					return nil
				}
			}
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "./keenctl.toml", "Path to the configuration file [default: ./keenctl.toml]")
	cmd.Flags().BoolVar(&dryRun, "dryRun", false, "Executes a dry run without changing routes")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Print debug information on stderr")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Silent mode")

	return cmd
}
