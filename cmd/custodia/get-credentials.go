// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "get credentials" subcommand for retrieving login/password pairs
// from the server and copying them to the system clipboard.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/spf13/cobra"
)

// getCredentialsCmd represents the "get credentials" command.
// It retrieves credentials (login and password) by name and optional version,
// displays metadata, and copies the credentials as JSON to the clipboard with a TTL.
var getCredentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "get credentials from custodia server",
	// RunE executes the command logic:
	//   - Read the credentials name (and optional version) from flags or interactively.
	//   - Create a client and a context with timeout.
	//   - Fetch the credentials and metadata from the server.
	//   - Marshal the credentials to JSON, display metadata, and copy the JSON to the clipboard.
	//   - The clipboard content will be cleared after SecretsTTL duration.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read the credentials name from flag or stdin.
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
		// Retrieve the credentials and metadata from the server.
		credentials, metadata, err := cli.GetCredentials(ctx, name, version)
		if err != nil {
			return fmt.Errorf("failed to get credentials: %w", err)
		}
		// Marshal credentials to JSON for clipboard and display.
		credentialsBytes, err := json.Marshal(credentials)
		if err != nil {
			return fmt.Errorf("failed to format credentials for console output: %w", err)
		}
		// Marshal metadata to JSON for display.
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to format metadata for console output: %w", err)
		}
		// Copy the credentials JSON to the clipboard with automatic cleanup.
		err = utils.CopyToClipboard(ctx, credentialsBytes, configuration.SecretsTTL.Duration())
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
	//   -n, --name     : credentials name (required, can be interactive)
	//   -v, --version  : credentials version (0 = latest, default 0)
	getCredentialsCmd.Flags().StringP("name", "n", "", "credentials name")
	getCredentialsCmd.Flags().Uint64P("version", "v", 0, "cred version")
	getCmd.AddCommand(getCredentialsCmd)
}
