package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login to custodia service",
	RunE: func(cmd *cobra.Command, args []string) error {
		login, err := cmd.Flags().GetString("login")
		if err != nil {
			return fmt.Errorf("failed to get login value from flag: %w", err)
		}

		if login == "" {
			fmt.Print("Please type your login: ")
			fmt.Scanln(&login)
		}
		fmt.Print("Please type your password: ")
		session.Password, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
		cli, configuration, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.Login(context.Background(), login, session.Password, configuration.SessionTTL.Duration()); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Println("logged in successfully")
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("login", "l", "", "your login for custodia service")
	rootCmd.AddCommand(loginCmd)
}
