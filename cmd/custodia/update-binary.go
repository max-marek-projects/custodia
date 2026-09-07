package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var updateSecretBinaryCmd = &cobra.Command{
	Use:   "binary",
	Short: "update secret binary data on custodia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return fmt.Errorf("failed to get name value from flag: %w", err)
		}
		if name == "" {
			fmt.Print("Please type cred name: ")
			fmt.Scanln(&name)
		}
		fmt.Print("Please type secret binary data: ")
		secretData, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
		cli, _, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.UpdateSecretBinary(context.Background(), name, secretData); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Printf("created cred `%s` successfully\n", name)
		return nil
	},
}

func init() {
	updateSecretBinaryCmd.Flags().StringP("name", "n", "", "secret data name")
	updateCmd.AddCommand(updateSecretBinaryCmd)
}
