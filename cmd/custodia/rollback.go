// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "rollback" subcommand for reverting a secret to its
// previous version. The command soft-deletes the latest active version, making
// the second‑latest version the current one.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// rollbackCmd represents the "rollback" command.
// It reverts a secret to the previous version by marking the current latest
// version as deleted. This operation requires at least two active versions
// of the secret; otherwise, an error is returned.
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "rollback secret data to its previous version",
	// RunE executes the command logic:
	//   - Read the secret name from flag or interactively.
	//   - Create a client and a context with timeout.
	//   - Call RollbackSecret to revert the secret.
	//   - Print a success message.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read secret name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
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
		// Rollback the secret to its previous version.
		err = cli.RollbackSecret(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to rollback secret: %w", err)
		}
		fmt.Println("successfully rolled secret data back")
		return nil
	},
}

// init registers the command with the root command and defines its flags.
func init() {
	// Flags:
	//   -n, --name : secret name (required, can be interactive)
	rollbackCmd.Flags().StringP("name", "n", "", "secret name")
	rootCmd.AddCommand(rollbackCmd)
}
