// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "update text" subcommand for updating the text content
// of an existing secret on the server.
package main

import (
	"errors"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// updateSecretTextCmd represents the "update text" command.
// It updates the text data of an existing secret by its name.
// The new text is read securely from the terminal.
// This operation creates a new version of the secret (does not overwrite the existing one).
var updateSecretTextCmd = &cobra.Command{
	Use:   "text",
	Short: "update secret text data on custodia server",
	// RunE executes the command logic:
	//   - Read the secret name from flag or interactively.
	//   - Read the new text data securely from the terminal (hidden input).
	//   - Create a client and a context with timeout.
	//   - Call UpdateSecretText to update the text content.
	//   - Print a success message.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read secret name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
		if err != nil {
			return err
		}
		// Read new text data securely (hidden input).
		secretData, err := client.ReadSecret("secret data")
		if err != nil {
			return err
		}
		// Create a new client and a context with timeout.
		cli, _, ctx, cancel, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cancel()
		defer cli.Close()
		// Update the secret text.
		if err := cli.UpdateSecretText(ctx, name, string(secretData)); err != nil {
			if errors.Is(err, client.ErrSecretTooLarge) {
				return fmt.Errorf("secret text is too large: %w", err)
			}
			return fmt.Errorf("failed to update secret text: %w", err)
		}
		fmt.Printf("updated secret text `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "update" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name : secret name (required, can be interactive)
	updateSecretTextCmd.Flags().StringP("name", "n", "", "secret data name")
	updateCmd.AddCommand(updateSecretTextCmd)
}
