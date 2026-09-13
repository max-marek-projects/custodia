// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "delete" subcommand for permanently removing a secret
// (soft-deletes all versions of a secret).
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// deleteCmd represents the "delete" command.
// It permanently soft-deletes all versions of a secret by its name.
// The secret will no longer be retrievable via get commands.
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "permanently delete a secret (all versions) from the server",
	// RunE executes the command logic.
	// It reads the secret name (interactively if not provided), creates a client,
	// and calls DeleteSecret to remove the secret.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read the secret name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
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
		// Delete the secret (soft-delete all versions).
		err = cli.DeleteSecret(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}
		fmt.Println("secret deleted successfully")
		return nil
	},
}

// init registers the command with the root command and defines its flags.
func init() {
	// Flags:
	//   -n, --name: secret name (required, can be interactive)
	deleteCmd.Flags().StringP("name", "n", "", "secret data name")
	rootCmd.AddCommand(deleteCmd)
}
