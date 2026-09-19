// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "logout" subcommand for terminating the current session
// on the current device or all devices.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

// logoutCmd represents the "logout" command.
// It revokes the refresh token for the current device (or all devices) on the server,
// clears the local session, and removes stored tokens.
// The command supports logging out from all devices using the --all flag.
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "logout from custodia service on current device",
	// RunE executes the command logic:
	//   - Check if the --all flag is set.
	//   - Create a client and a context with timeout.
	//   - Call Logout with the allDevices flag to revoke tokens.
	//   - Clear the local session and token storage.
	//   - Print a success message.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read the --all flag (default false).
		allDevices, err := cmd.Flags().GetBool("all")
		if err != nil {
			return fmt.Errorf("failed to get `all` value from flag: %w", err)
		}
		// Create a new client and a context with timeout.
		cli, _, ctx, cancel, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cancel()
		defer cli.Close()
		// Perform logout.
		if err := cli.Logout(ctx, allDevices); err != nil {
			return fmt.Errorf("failed to logout: %w", err)
		}
		fmt.Println("logged out successfully")
		return nil
	},
}

// init registers the command with the root command and defines its flags.
func init() {
	// Flags:
	//   -a, --all : revoke tokens on all devices (default false).
	logoutCmd.Flags().BoolP("all", "a", false, "logout from all devices")
	rootCmd.AddCommand(logoutCmd)
}
