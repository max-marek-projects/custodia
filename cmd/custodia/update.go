// Package main provides the CLI commands for the Custodia password manager.
// This file defines the parent command for all "update" subcommands.
package main

import "github.com/spf13/cobra"

// updateCmd is the parent command for updating existing secrets on the server.
// It groups all subcommands that update different types of secret data:
//   - binary      : arbitrary binary data
//   - card-data   : bank card details
//   - credentials : login/password pairs
//   - metadata    : key-value metadata (without changing the secret data)
//   - text        : arbitrary text data
//
// All update operations create a new version of the secret, preserving history.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update secret item on server",
}

// init registers the update command with the root command.
func init() {
	rootCmd.AddCommand(updateCmd)
}
