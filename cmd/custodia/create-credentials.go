package main

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addCredentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "add new credentials to custodia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return fmt.Errorf("failed to get name value from flag: %w", err)
		}
		login, err := cmd.Flags().GetString("login")
		if err != nil {
			return fmt.Errorf("failed to get login value from flag: %w", err)
		}
		rawMetadata, err := cmd.Flags().GetString("meta")
		if err != nil {
			return fmt.Errorf("failed to get metadata from flag: %w", err)
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
		metadata, err := parseMeta(rawMetadata)
		if err != nil {
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
		cli, _, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.CreateCredentials(context.Background(), name, &models.Credentials{Login: login, Password: string(password)}, metadata); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Printf("created cred %s successfully\n", name)
		return nil
	},
}

func parseMeta(pairs string) (map[string]string, error) {
	var metadata map[string]string
	err := json.Unmarshal([]byte(pairs), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	return metadata, nil
}

func init() {
	addCredentialsCmd.Flags().StringP("name", "n", "", "credentials name")
	addCredentialsCmd.Flags().StringP("meta", "m", "{}", "credentials metadata")
	addCredentialsCmd.Flags().StringP("login", "l", "", "login to store in custodia server")
	createCmd.AddCommand(addCredentialsCmd)
}
