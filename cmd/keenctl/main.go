// Binary routeek
package main

import (
	"log/slog"
	"os"

	"github.com/docopt/docopt-go"
	app "github.com/idr0id/keenctl/internal/app"
)

const usage = `keenctl - manage static routes on Keenetic's router.

Usage:
  keenctl [options] [-v]...

Options:
  -c --config <path>      Path to the configuration file.
                           [default: ./config.toml]
  -n --dry-run            Executes a dry run without changing routes.
  -v --verbose            Print debug information on stderr.
  -q --quiet              Silent mode.
  -h --help               Show this help.
`

var version = "[dev-build]"

func main() {
	var (
		args          = parseArgs()
		logger        = setupLogger(args)
		configPath, _ = args["--config"].(string)
		dryRun, _     = args["--dry-run"].(bool)
	)

	logger.Info("parse configuration")
	conf, err := app.ParseConfig(configPath, dryRun)
	if err != nil {
		logger.Error("configuration error", slog.Any("error", err))
		os.Exit(1)
	}

	a := app.New(conf, logger)
	if err := a.Run(); err != nil {
		logger.Error("fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}

func parseArgs() map[string]interface{} {
	args, err := docopt.ParseArgs(usage, nil, version)
	if err != nil {
		docopt.PrintHelpAndExit(err, usage)
	}

	return args
}
