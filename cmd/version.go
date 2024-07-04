package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Displays the current version of keenctl",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("keenctl version v0.1.0")
		},
	}
}
