// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "update credentials" subcommand for updating an existing
// login/password pair on the server.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/spf13/cobra"
)

// updateCredentialsCmd represents the "update credentials" command.
// It updates the login and password of an existing secret by its name.
// The new password is read securely from the terminal, and the login is read
// from a flag or interactively.
// This operation creates a new version of the secret (does not overwrite the existing one).
var updateCredentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "update credentials on custodia server",
	// RunE executes the command logic:
	//   - Read the credentials name from flag or interactively.
	//   - Read the login from flag or interactively.
	//   - Read the new password securely from the terminal (hidden input).
	//   - Create a client and a context with timeout.
	//   - Call UpdateCredentials to update the credentials.
	//   - Print a success message.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read the credentials name from flag or stdin.
		name, err := client.ReadStringValue(cmd, "name")
		if err != nil {
			return err
		}
		// Read the login from flag or stdin.
		login, err := client.ReadStringValue(cmd, "login")
		if err != nil {
			return err
		}
		// Read the new password securely (hidden input).
		password, err := client.ReadSecret("password")
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
		// Update the credentials.
		if err := cli.UpdateCredentials(ctx, name, &models.Credentials{
			Login:    login,
			Password: string(password),
		}); err != nil {
			return fmt.Errorf("failed to update credentials: %w", err)
		}
		fmt.Printf("updated credentials `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "update" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name : credentials name (required, can be interactive)
	//   -l, --login : new login (optional, can be interactive)
	updateCredentialsCmd.Flags().StringP("name", "n", "", "credentials name")
	updateCredentialsCmd.Flags().StringP("login", "l", "", "login to store in custodia server")
	updateCmd.AddCommand(updateCredentialsCmd)
}
