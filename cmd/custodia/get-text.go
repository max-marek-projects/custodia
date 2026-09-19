// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "get text" subcommand for retrieving arbitrary text secrets
// from the server and copying them to the system clipboard.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/spf13/cobra"
)

// getSecretTextCmd represents the "get text" command.
// It retrieves a text secret by name and optional version,
// displays its metadata, and copies the secret text to the clipboard with a TTL.
var getSecretTextCmd = &cobra.Command{
	Use:   "text",
	Short: "get secret text from custodia server",
	// RunE executes the command logic:
	//   - Read the secret name (and optional version) from flags or interactively.
	//   - Create a client and a context with timeout.
	//   - Fetch the secret text and metadata from the server.
	//   - Display metadata and copy the secret text to the clipboard.
	//   - The clipboard content will be cleared after SecretsTTL duration.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read the secret name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
		if err != nil {
			return err
		}
		// Read version from flag (default 0 = latest).
		version, err := cmd.Flags().GetUint64("version")
		if err != nil {
			return fmt.Errorf("failed to get version value from flag: %w", err)
		}
		fmt.Println()
		// Create a new client and a context with timeout.
		cli, configuration, ctx, cancel, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cancel()
		defer cli.Close()
		// Retrieve the secret text and metadata from the server.
		secretText, metadata, err := cli.GetSecretText(ctx, name, version)
		if err != nil {
			return fmt.Errorf("failed to get secret text: %w", err)
		}
		// Marshal metadata to JSON for display.
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to format metadata for console output: %w", err)
		}
		// Copy the secret text to the clipboard with automatic cleanup.
		err = utils.CopyToClipboard(ctx, []byte(secretText), configuration.SecretsTTL.Duration())
		if err != nil {
			return fmt.Errorf("failed to save secret data to clipboard: %w", err)
		}
		fmt.Printf("received secret data and stored to clipboard. will be removed after: %s\n", configuration.SecretsTTL)
		fmt.Printf("received metadata: %s\n", string(metadataBytes))
		return nil
	},
}

// init registers the command with the parent "get" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name     : secret name (required, can be interactive)
	//   -v, --version  : secret version (0 = latest, default 0)
	getSecretTextCmd.Flags().StringP("name", "n", "", "secret text data name")
	getSecretTextCmd.Flags().Uint64P("version", "v", 0, "cred version")
	getCmd.AddCommand(getSecretTextCmd)
}
