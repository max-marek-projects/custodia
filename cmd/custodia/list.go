// Package main provides the CLI commands for the Custodia password manager.
// This file implements the "list" subcommand for listing secrets, optionally
// filtered by metadata.
package main

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/spf13/cobra"
)

// listCmd represents the "list" command.
// It lists all user secrets, optionally filtered by metadata.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list secrets, optionally filtered by metadata",
	// RunE executes the command logic:
	//   - Read optional metadata filter from flag.
	//   - Create a client and a context with timeout.
	//   - Call ListSecrets and print the results as a table.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read filter filter (optional).
		filter, err := client.ReadMapValue(cmd, "metadata")
		if err != nil {
			return err
		}

		cli, _, ctx, cancel, err := client.NewClient(session)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cancel()
		defer cli.Close()

		secrets, err := cli.ListSecrets(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list secrets: %w", err)
		}

		if len(secrets) == 0 {
			fmt.Println("no secrets found")
			return nil
		}

		fmt.Printf("%-30s %-15s %-10s %s\n", "NAME", "TYPE", "VERSION", "METADATA")
		for _, s := range secrets {
			metaJSON := utils.OrNA(fmt.Sprintf("%v", s.Metadata))
			fmt.Printf("%-30s %-15s %-10d %s\n", s.Name, s.Type, s.LatestVersion, metaJSON)
		}
		return nil
	},
}

// init registers the command with the root command and defines its flags.
func init() {
	// Flags:
	//   -m, --metadata : optional metadata filter as JSON (e.g. '{"env":"prod"}')
	listCmd.Flags().StringP("metadata", "m", "", "metadata filter as JSON")
	rootCmd.AddCommand(listCmd)
}
