// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "create binary" subcommand for storing arbitrary binary data.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// addSecretBinaryCmd represents the "create binary" command.
// It reads a secret name, metadata, and binary data from the user,
// then encrypts and sends it to the server for storage.
var addSecretBinaryCmd = &cobra.Command{
	Use:   "binary",
	Short: "add new secret binary data to custodia server",
	// RunE executes the command logic.
	// It prompts for missing flags, creates a client, and calls CreateSecretBinary.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read the secret name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
		if err != nil {
			return err
		}
		// Read metadata (JSON) from flag or stdin.
		metadata, err := client.ReadMapValue(cmd, "metadata")
		if err != nil {
			return err
		}
		// Read binary data securely from terminal (hidden input).
		secretData, err := client.ReadSecret("secret binary data")
		if err != nil {
			return err
		}
		// Create a new client and a context with timeout.
		cli, _, ctx, cancel, err := client.NewClient(session)
		defer cancel()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		// Send the encrypted secret to the server.
		if err := cli.CreateSecretBinary(ctx, name, secretData, metadata); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Printf("created secret binary data `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "create" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name: secret name (required, can be interactive)
	//   -m, --metadata: JSON metadata (default: "{}")
	addSecretBinaryCmd.Flags().StringP("name", "n", "", "secret data name")
	addSecretBinaryCmd.Flags().StringP("metadata", "m", "{}", "secret data metadata")
	createCmd.AddCommand(addSecretBinaryCmd)
}
