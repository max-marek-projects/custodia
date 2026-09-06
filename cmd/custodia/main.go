// Package main implements a simple CLI application for a password management server.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// rootCmd implements root command
var rootCmd = &cobra.Command{
	Use:          "custodia",
	Short:        "Custodia - simple but secure password manager",
	SilenceUsage: true,
}
