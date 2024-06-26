package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Displays the current version of keenctl",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("keenctl version v0.1.0")
		},
	}
}
