// Package main is the entry point for the keenctl command-line application.
// It sets up and executes the commands defined in the cmd package.
package main

import (
	"github.com/idr0id/keenctl/cmd"
)

func main() {
	cmd.Execute()
}
