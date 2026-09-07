package main

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

var updateMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "update secret metadata on custodia server",
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
		metadata, err := parseMeta(rawMetadata)
		if err != nil {
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
		cli, _, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.UpdateMetadata(context.Background(), name, metadata); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Printf("updated metadata for cred `%s` successfully\n", name)
		return nil
	},
}

func init() {
	updateMetadataCmd.Flags().StringP("name", "n", "", "secret data name")
	updateMetadataCmd.Flags().StringP("meta", "m", "{}", "secret data metadata")
	updateCmd.AddCommand(updateMetadataCmd)
}
