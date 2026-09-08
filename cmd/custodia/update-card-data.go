// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "update card-data" subcommand for updating existing
// bank card details on the server.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/spf13/cobra"
)

// updateCardDataCmd represents the "update card-data" command.
// It updates the card details (number, holder, expiration, CVV) of an existing
// secret by its name. All card data is read securely from the terminal.
// This operation creates a new version of the secret (does not overwrite the existing one).
var updateCardDataCmd = &cobra.Command{
	Use:   "card-data",
	Short: "update card data on custodia server",
	// RunE executes the command logic:
	//   - Read the card data name from flag or interactively.
	//   - Read the card number, holder name, expiration date, and CVV securely.
	//   - Create a client and a context with timeout.
	//   - Call UpdateCardData to update the card data.
	//   - Print a success message.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read card data name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
		if err != nil {
			return err
		}
		// Read all card details securely (hidden input).
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
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cancel()
		defer cli.Close()
		// Update the card data.
		cardData, err := models.NewCardData(
			string(cardNumber),
			string(holderName),
			string(expirationDate),
			string(cvv),
		)
		if err != nil {
			return fmt.Errorf("wrong card data: %w", err)
		}
		if err := cli.UpdateCardData(ctx, name, cardData); err != nil {
			return fmt.Errorf("failed to update card data: %w", err)
		}
		fmt.Printf("updated card data `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "update" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name : card data name (required, can be interactive)
	updateCardDataCmd.Flags().StringP("name", "n", "", "card data name")
	updateCmd.AddCommand(updateCardDataCmd)
}
