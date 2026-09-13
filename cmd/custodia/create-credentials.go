// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "create credentials" subcommand for storing login/password pairs.
package main

import (
	"errors"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/spf13/cobra"
)

// addCredentialsCmd represents the "create credentials" command.
// It collects a name, login, password, and optional metadata from the user,
// then encrypts and stores the credentials on the server.
var addCredentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "add new credentials to custodia server",
	// RunE executes the command logic.
	// It prompts for missing flags, reads the password securely, creates a client,
	// and calls CreateCredentials.
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
		// Read metadata (JSON) from flag or stdin.
		metadata, err := client.ReadMapValue(cmd, "metadata")
		if err != nil {
			return err
		}
		// Read the password securely (hidden input).
		password, err := client.ReadSecret("password")
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
		// Send the encrypted credentials to the server.
		if err := cli.CreateCredentials(ctx, name, &models.Credentials{
			Login:    login,
			Password: string(password),
		}, metadata); err != nil {
			if errors.Is(err, client.ErrSecretTooLarge) {
				return fmt.Errorf("credentials data is too large: %w", err)
			}
			return fmt.Errorf("failed to create credentials: %w", err)
		}
		fmt.Printf("created credentials `%s` successfully\n", name)
		return nil
	},
}

// init registers the command with the parent "create" command
// and defines its command-line flags.
func init() {
	// Flags:
	//   -n, --name: credentials name (required, can be interactive)
	//   -l, --login: login to store (required, can be interactive)
	//   -m, --metadata: JSON metadata (default: "{}")
	addCredentialsCmd.Flags().StringP("name", "n", "", "credentials name")
	addCredentialsCmd.Flags().StringP("metadata", "m", "{}", "credentials metadata")
	addCredentialsCmd.Flags().StringP("login", "l", "", "login to store in custodia server")
	createCmd.AddCommand(addCredentialsCmd)
}
