// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "update metadata" subcommand for updating the metadata
// of an existing secret on the server.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// updateMetadataCmd represents the "update metadata" command.
// It updates the metadata (key-value pairs) of an existing secret by its name.
// The new metadata is read as a JSON object from a flag or interactively.
// This operation creates a new version of the secret (does not overwrite the existing one).
var updateMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "update secret metadata on custodia server",
	// RunE executes the command logic:
	//   - Read the secret name from flag or interactively.
	//   - Read the metadata as a JSON object from flag or interactively.
	//   - Create a client and a context with timeout.
	//   - Call UpdateMetadata to update the secret metadata.
	//   - Print a success message.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read secret name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
		if err != nil {
			return err
		}
		// Read metadata as JSON from flag or stdin.
		metadata, err := client.ReadMapValue(cmd, "metadata")
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
		// Update the metadata.
		if err := cli.UpdateMetadata(ctx, name, metadata); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
		fmt.Printf("updated metadata for secret `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "update" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name     : secret name (required, can be interactive)
	//   -m, --metadata : JSON object with metadata (default: "{}", can be interactive)
	updateMetadataCmd.Flags().StringP("name", "n", "", "secret data name")
	updateMetadataCmd.Flags().StringP("metadata", "m", "{}", "secret data metadata")
	updateCmd.AddCommand(updateMetadataCmd)
}
