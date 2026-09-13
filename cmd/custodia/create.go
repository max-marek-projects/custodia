// Package main provides the CLI commands for the Custodia password manager.
// This file defines the parent command for all "create" subcommands.
package main

import "github.com/spf13/cobra"

// createCmd is the parent command for creating new secrets on the server.
// It groups all subcommands that add different types of secret data:
//   - binary    : arbitrary binary data
//   - card-data : bank card details
//   - credentials : login/password pairs
//   - text      : arbitrary text data
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create new secret item on server",
}

// init registers the create command with the root command.
func init() {
	rootCmd.AddCommand(createCmd)
}
