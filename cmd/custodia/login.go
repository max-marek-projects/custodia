// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "login" subcommand for authenticating a user
// and establishing a session with the server.
package main

import (
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// loginCmd represents the "login" command.
// It authenticates the user with the server using a login and password,
// creates a local session, and stores authentication tokens for subsequent requests.
// The session TTL is determined by the client configuration.
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login to custodia service",
	// RunE executes the command logic:
	//   - Read the login from flag or interactively.
	//   - Read the password securely from the terminal (hidden input).
	//   - Create a client and a context with timeout.
	//   - Call Login to authenticate and obtain tokens.
	//   - The tokens are saved locally for future API calls.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read login from flag or stdin.
		login, err := client.ReadStringValue(cmd, "login")
		if err != nil {
			return err
		}
		// Read password securely (hidden input).
		fmt.Print("Please type your password: ")
		session.Password, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
		// Create a new client and a context with timeout.
		cli, configuration, ctx, cancel, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cancel()
		defer cli.Close()
		// Perform login.
		if err := cli.Login(ctx, login, session.Password, configuration.SessionTTL.Duration()); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Println("logged in successfully")
		return nil
	},
}

// init registers the command with the root command and defines its flags.
func init() {
	// Flags:
	//   -l, --login : username for login (optional; if not provided, user will be prompted)
	loginCmd.Flags().StringP("login", "l", "", "your login for custodia service")
	rootCmd.AddCommand(loginCmd)
}
