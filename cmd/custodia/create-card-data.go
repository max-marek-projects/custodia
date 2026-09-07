package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addCardDataCmd = &cobra.Command{
	Use:   "card-data",
	Short: "add new card-data to custodia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return fmt.Errorf("failed to get name value from flag: %w", err)
		}
		rawMetadata, err := cmd.Flags().GetString("meta")
		if err != nil {
			return fmt.Errorf("failed to get metadata from flag: %w", err)
		}
		if name == "" {
			fmt.Print("Please type cred name: ")
			fmt.Scanln(&name)
		}
		fmt.Print("Please type card number in format `0000 0000 0000 0000`: ")
		cardNumber, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read card number: %w", err)
		}
		fmt.Println()
		fmt.Print("Please type holder name: ")
		holderName, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read holder name: %w", err)
		}
		fmt.Println()
		fmt.Print("Please type expiration date in format 01/2001: ")
		expirationData, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read expiration date: %w", err)
		}
		fmt.Println()
		fmt.Print("Please type cvv: ")
		cvv, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read cvv: %w", err)
		}
		fmt.Println()
		metadata, err := parseMeta(rawMetadata)
		if err != nil {
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
		cli, _, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.CreateCardData(context.Background(), name, &models.CardData{Number: string(cardNumber), HolderName: string(holderName), ExpirationDate: string(expirationData), CVV: string(cvv)}, metadata); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Printf("created card data `%s` successfully\n", name)
		return nil
	},
}

func init() {
	addCardDataCmd.Flags().StringP("name", "n", "", "card data name")
	addCardDataCmd.Flags().StringP("meta", "m", "{}", "card data metadata")
	createCmd.AddCommand(addCardDataCmd)
}
