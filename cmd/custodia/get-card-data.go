// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "get card-data" subcommand for retrieving bank card details
// from the server and copying them to the system clipboard.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/spf13/cobra"
)

// getCardDataCmd represents the "get card-data" command.
// It retrieves card data (number, holder, expiration, CVV) by name and optional version,
// displays its metadata, and copies the card data as JSON to the clipboard with a TTL.
var getCardDataCmd = &cobra.Command{
	Use:   "card-data",
	Short: "get card-data from custodia server",
	// RunE executes the command logic:
	//   - Read the card data name (and optional version) from flags or interactively.
	//   - Create a client and a context with timeout.
	//   - Fetch the card data and metadata from the server.
	//   - Marshal the card data to JSON, display metadata, and copy the JSON to the clipboard.
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
		// Retrieve the card data and metadata from the server.
		cardData, metadata, err := cli.GetCardData(ctx, name, version)
		if err != nil {
			return fmt.Errorf("failed to get card data: %w", err)
		}
		// Marshal card data to JSON for clipboard and display.
		cardDataBytes, err := json.Marshal(cardData)
		if err != nil {
			return fmt.Errorf("failed to format card data for console output: %w", err)
		}
		// Marshal metadata to JSON for display.
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to format metadata for console output: %w", err)
		}
		// Copy the card data JSON to the clipboard with automatic cleanup.
		err = utils.CopyToClipboard(ctx, cardDataBytes, configuration.SecretsTTL.Duration())
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
	//   -n, --name     : card data name (required, can be interactive)
	//   -v, --version  : card version (0 = latest, default 0)
	getCardDataCmd.Flags().StringP("name", "n", "", "credentials name")
	getCardDataCmd.Flags().Uint64P("version", "v", 0, "cred version")
	getCmd.AddCommand(getCardDataCmd)
}
