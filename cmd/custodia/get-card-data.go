package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/spf13/cobra"
)

var getCardDataCmd = &cobra.Command{
	Use:   "card-data",
	Short: "get card-data from custodia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return fmt.Errorf("failed to get name value from flag: %w", err)
		}
		version, err := cmd.Flags().GetUint64("version")
		if err != nil {
			return fmt.Errorf("failed to get version value from flag: %w", err)
		}
		if name == "" {
			fmt.Print("Please type secret data name: ")
			fmt.Scanln(&name)
		}
		fmt.Println()
		cli, configuration, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		ctx, cancel := context.WithTimeout(context.Background(), configuration.RequestTimeout.Duration())
		defer cancel()
		cardData, metadata, err := cli.GetCardData(ctx, name, version)
		if err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		cardDataBytes, err := json.Marshal(cardData)
		if err != nil {
			return fmt.Errorf("failed to plot credentials to console: %w", err)
		}
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to plot metadata to console: %w", err)
		}
		err = utils.CopyToClipboard(ctx, cardDataBytes, configuration.SecretsTTL.Duration())
		if err != nil {
			return fmt.Errorf("failed to save secret data to clipboard: %w", err)
		}
		fmt.Printf("received secret data and stored to clipboard. will be removed after: %s\n", configuration.SecretsTTL)
		fmt.Printf("received metadata: %s\n", string(metadataBytes))
		return nil
	},
}

func init() {
	getCardDataCmd.Flags().StringP("name", "n", "", "credentials name")
	getCardDataCmd.Flags().Uint64P("version", "v", 0, "cred version")
	getCmd.AddCommand(getCardDataCmd)
}
