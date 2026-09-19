// Package main provides the CLI commands for the Custodia password manager.
// This file defines the parent command for all "get" subcommands.
package main

import "github.com/spf13/cobra"

// getCmd is the parent command for retrieving secrets from the server.
// It groups all subcommands that fetch different types of secret data:
//   - binary    : arbitrary binary data
//   - card-data : bank card details
//   - credentials : login/password pairs
//   - text      : arbitrary text data
//
// Retrieved secrets are copied to the system clipboard with a configurable TTL.
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get secret item from server",
}

// init registers the get command with the root command.
func init() {
	rootCmd.AddCommand(getCmd)
}
