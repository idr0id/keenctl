package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var silentErr = errors.New("SilentErr")

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "keenctl",
		Short: "keenctl is a utility for managing a static routes for Keenetic routers.",
		Long: `keenctl is a command-line utility designed to manage static routes on Keenetic routers. 
    		It provides various features including SSH remote access configuration, DNS and ASN address resolution, 
    		and filtering of private or unspecified addresses.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return silentErr
	})
	rootCmd.AddCommand(newServeCommand())
	rootCmd.AddCommand(newVersionCommand())

	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, silentErr) {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
