// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "update binary" subcommand for updating the binary data
// of an existing secret on the server.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// updateSecretBinaryCmd represents the "update binary" command.
// It updates the binary data of an existing secret by its name.
// The new binary data is read securely from the terminal.
// This operation creates a new version of the secret (does not overwrite the existing one).
var updateSecretBinaryCmd = &cobra.Command{
	Use:   "binary",
	Short: "update secret binary data on custodia server",
	// RunE executes the command logic:
	//   - Read the secret name from flag or interactively.
	//   - Read the new binary data securely from the terminal (hidden input).
	//   - Create a client and a context with timeout.
	//   - Call UpdateSecretBinary to update the secret data.
	//   - Print a success message.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read secret name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
		if err != nil {
			return err
		}
		// Read new binary data securely (hidden input).
		secretData, err := client.ReadSecret("secret binary data")
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
		// Update the secret binary data.
		if err := cli.UpdateSecretBinary(ctx, name, secretData); err != nil {
			return fmt.Errorf("failed to update secret binary data: %w", err)
		}
		fmt.Printf("updated secret binary data `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "update" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name : secret name (required, can be interactive)
	updateSecretBinaryCmd.Flags().StringP("name", "n", "", "secret data name")
	updateCmd.AddCommand(updateSecretBinaryCmd)
}
