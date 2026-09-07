package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var updateCredentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "update credentials on custodia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return fmt.Errorf("failed to get name value from flag: %w", err)
		}
		login, err := cmd.Flags().GetString("login")
		if err != nil {
			return fmt.Errorf("failed to get login value from flag: %w", err)
		}
		if name == "" {
			fmt.Print("Please type cred name: ")
			fmt.Scanln(&name)
		}
		if login == "" {
			fmt.Print("Please type login: ")
			fmt.Scanln(&login)
		}
		fmt.Print("Please type password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
		cli, _, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.UpdateCredentials(context.Background(), name, &models.Credentials{Login: login, Password: string(password)}); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Printf("created cred %s successfully\n", name)
		return nil
	},
}

func init() {
	updateCredentialsCmd.Flags().StringP("name", "n", "", "credentials name")
	updateCredentialsCmd.Flags().StringP("login", "l", "", "login to store in custodia server")
	updateCmd.AddCommand(updateCredentialsCmd)
}
