// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "create text" subcommand for storing arbitrary text data.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// addSecretTextCmd represents the "create text" command.
// It collects a secret name, metadata, and text data from the user,
// then encrypts and stores the text on the server.
var addSecretTextCmd = &cobra.Command{
	Use:   "text",
	Short: "add new secret text data to custodia server",
	// RunE executes the command logic.
	// It prompts for missing flags, reads text securely, creates a client,
	// and calls CreateSecretText.
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
		// Read the text data securely (hidden input).
		secretData, err := client.ReadSecret("secret data")
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
		// Send the encrypted text to the server.
		if err := cli.CreateSecretText(ctx, name, string(secretData), metadata); err != nil {
			return fmt.Errorf("failed to create secret text: %w", err)
		}
		fmt.Printf("created secret text data `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "create" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name: secret name (required, can be interactive)
	//   -m, --metadata: JSON metadata (default: "{}")
	addSecretTextCmd.Flags().StringP("name", "n", "", "secret data name")
	addSecretTextCmd.Flags().StringP("metadata", "m", "{}", "secret data metadata")
	createCmd.AddCommand(addSecretTextCmd)
}
