// Package main implements a simple CLI application for a password management server.

package main

import (
	"fmt"
	"os"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

func main() {
	if len(os.Args) > 1 {
		if err := rootCmd.Execute(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}
	if err := client.RunREPL(rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// rootCmd implements root command
var rootCmd = &cobra.Command{
	Use:          "custodia",
	Short:        "Custodia - simple but secure password manager",
	SilenceUsage: true,
}
var session = &client.Session{}
