// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "register" subcommand for creating a new user account
// on the Custodia server.
package main

import (
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// registerCmd represents the "register" command.
// It creates a new user account on the server using the provided login and password,
// then automatically logs the user in, establishes a session, and stores tokens locally.
// The session TTL is determined by the client configuration.
var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "register to custodia service",
	// RunE executes the command logic:
	//   - Read the login from flag or interactively.
	//   - Read the password securely from the terminal (hidden input).
	//   - Create a client and a context with timeout.
	//   - Call Register to create the account and obtain tokens.
	//   - The tokens are saved locally, and the session is established.
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
			return err
		}
		fmt.Println()
		// Create a new client and a context with timeout.
		cli, configuration, ctx, cancel, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cancel()
		defer cli.Close()
		// Register the new user.
		if err := cli.Register(ctx, login, session.Password, configuration.SessionTTL.Duration()); err != nil {
			return err
		}
		fmt.Println("registered successfully")
		return nil
	},
}

// init registers the command with the root command and defines its flags.
func init() {
	// Flags:
	//   -l, --login : desired username for registration (optional; if not provided, user will be prompted)
	registerCmd.Flags().StringP("login", "l", "", "your login for custodia service")
	rootCmd.AddCommand(registerCmd)
}
