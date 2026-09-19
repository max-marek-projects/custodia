// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "refresh" subcommand for obtaining a new access token
// using a valid refresh token stored locally.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// refreshCmd represents the "refresh" command.
// It uses the stored refresh token to request a new access token from the server,
// updates the local token storage, and keeps the session active without requiring
// the user to re-enter credentials.
var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "refresh access to custodia server",
	// RunE executes the command logic:
	//   - Create a client and a context with timeout.
	//   - Call RefreshAccess to obtain a new access token.
	//   - The new token is automatically saved to local storage.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a new client and a context with timeout.
		cli, _, ctx, cancel, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cancel()
		defer cli.Close()
		// Refresh the access token.
		if err := cli.RefreshAccess(ctx); err != nil {
			return err
		}
		fmt.Println("refreshed access successfully")
		return nil
	},
}

// init registers the refresh command with the root command.
func init() {
	rootCmd.AddCommand(refreshCmd)
}
