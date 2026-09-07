package main

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "refresh access to custodia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cli, _, err := client.NewClient(session)
		if err != nil {
			return err
		}
		defer cli.Close()
		if err := cli.RefreshAccess(context.Background()); err != nil {
			return err
		}
		fmt.Println("refreshed access successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}
