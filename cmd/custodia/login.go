package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/config"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login to custodia service",
	RunE: func(cmd *cobra.Command, args []string) error {
		configuration, err := config.NewClientConf()
		if err != nil {
			return fmt.Errorf("failed to initialize configuration: %w", err)
		}
		err = logger.Initialize(configuration.LoggerLevel)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		login, err := cmd.Flags().GetString("login")
		if err != nil {
			return fmt.Errorf("failed to get login value from flag: %w", err)
		}
		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return fmt.Errorf("failed to get password value from flag: %w", err)
		}

		if login == "" {
			fmt.Print("Please type your login: ")
			fmt.Scanln(&login)
		}
		if password == "" {
			fmt.Print("Please type your password: ")
			bytePass, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password = string(bytePass)
			fmt.Println()
		}
		cli, err := client.NewClient(configuration.ServerAddr, configuration.ConfigFolder, configuration.TokenFilename)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.Login(context.Background(), login, password); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Println("logged in successfully")
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("login", "l", "", "your login for custodia service")
	loginCmd.Flags().StringP("password", "p", "", "your password for custodia service")
	rootCmd.AddCommand(loginCmd)
}
