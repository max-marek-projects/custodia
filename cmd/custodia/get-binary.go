// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "get binary" subcommand for retrieving binary secrets
// from the server and copying them to the system clipboard.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/spf13/cobra"
)

// getSecretBinaryCmd represents the "get binary" command.
// It retrieves a binary secret by name and optional version, displays its metadata,
// and copies the secret data to the clipboard with a configurable TTL.
var getSecretBinaryCmd = &cobra.Command{
	Use:   "binary",
	Short: "get secret binary data from custodia server",
	// RunE executes the command logic:
	//   - Read the secret name (and version) from flags or interactively.
	//   - Create a client and a context with timeout.
	//   - Fetch the secret binary data and metadata from the server.
	//   - Display the metadata as JSON and copy the data to the clipboard.
	//   - The clipboard content will be cleared after SecretsTTL duration.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read secret name from flag or stdin.
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
		// Retrieve the secret data and metadata from the server.
		secretData, metadata, err := cli.GetSecretBinary(ctx, name, version)
		if err != nil {
			return fmt.Errorf("failed to get binary secret: %w", err)
		}
		// Format metadata as JSON for display.
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to format metadata for console output: %w", err)
		}
		// Copy the secret data to the clipboard with automatic cleanup.
		err = utils.CopyToClipboard(ctx, secretData, configuration.SecretsTTL.Duration())
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
	getSecretBinaryCmd.Flags().StringP("name", "n", "", "secret text data name")
	getSecretBinaryCmd.Flags().Uint64P("version", "v", 0, "cred version")
	getCmd.AddCommand(getSecretBinaryCmd)
}
