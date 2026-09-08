// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "create card-data" subcommand for storing bank card details.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/spf13/cobra"
)

// addCardDataCmd represents the "create card-data" command.
// It collects card number, holder name, expiration date, and CVV from the user,
// then encrypts and stores the card data on the server.
var addCardDataCmd = &cobra.Command{
	Use:   "card-data",
	Short: "add new card-data to custodia server",
	// RunE executes the command logic.
	// It prompts for missing flags and card details, creates a client,
	// and calls CreateCardData.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read the card data name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
		if err != nil {
			return err
		}
		// Read metadata (JSON) from flag or stdin.
		metadata, err := client.ReadMapValue(cmd, "metadata")
		if err != nil {
			return err
		}
		// Read card details securely (hidden input).
		cardNumber, err := client.ReadSecret("card number in format `0000 0000 0000 0000`")
		if err != nil {
			return err
		}
		holderName, err := client.ReadSecret("holder name")
		if err != nil {
			return err
		}
		expirationDate, err := client.ReadSecret("expiration date in format 01/2001")
		if err != nil {
			return err
		}
		cvv, err := client.ReadSecret("cvv")
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
		// Send the encrypted card data to the server.
		cardData, err := models.NewCardData(
			string(cardNumber),
			string(holderName),
			string(expirationDate),
			string(cvv),
		)
		if err != nil {
			return fmt.Errorf("wrong card data: %w", err)
		}
		if err := cli.CreateCardData(ctx, name, cardData, metadata); err != nil {
			return fmt.Errorf("failed to create card data: %w", err)
		}
		fmt.Printf("created card data `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "create" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name: card data name (required, can be interactive)
	//   -m, --metadata: JSON metadata (default: "{}")
	addCardDataCmd.Flags().StringP("name", "n", "", "card data name")
	addCardDataCmd.Flags().StringP("metadata", "m", "{}", "card data metadata")
	createCmd.AddCommand(addCardDataCmd)
}
