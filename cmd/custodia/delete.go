package main

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete secret data to it's previous version",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return fmt.Errorf("failed to get name value from flag: %w", err)
		}
		if name == "" {
			fmt.Print("Please type cred name: ")
			fmt.Scanln(&name)
		}
		fmt.Println()
		cli, configuration, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to delete client: %w", err)
		}
		defer cli.Close()
		ctx, cancel := context.WithTimeout(context.Background(), configuration.RequestTimeout.Duration())
		defer cancel()
		err = cli.RollbackSecret(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}
		fmt.Printf("successfully deleted secret data back")
		return nil
	},
}

func init() {
	deleteCmd.Flags().StringP("name", "n", "", "secret data name")
	rootCmd.AddCommand(deleteCmd)
}
