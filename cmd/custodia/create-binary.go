package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addSecretBinaryCmd = &cobra.Command{
	Use:   "text",
	Short: "add new secret text data to custodia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return fmt.Errorf("failed to get name value from flag: %w", err)
		}
		rawMetadata, err := cmd.Flags().GetString("meta")
		if err != nil {
			return fmt.Errorf("failed to get metadata from flag: %w", err)
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
		metadata, err := parseMeta(rawMetadata)
		if err != nil {
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
		cli, _, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.CreateSecretBinary(context.Background(), name, secretData, metadata); err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
		fmt.Printf("created cred `%s` successfully\n", name)
		return nil
	},
}

func init() {
	addSecretBinaryCmd.Flags().StringP("name", "n", "", "secret data name")
	addSecretBinaryCmd.Flags().StringP("meta", "m", "{}", "secret data metadata")
	createCmd.AddCommand(addSecretBinaryCmd)
}
