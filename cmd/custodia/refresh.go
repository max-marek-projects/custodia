package main

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/config"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "refresh access to custodia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		configuration, err := config.NewClientConf()
		if err != nil {
			return err
		}
		err = logger.Initialize(configuration.LoggerLevel)
		if err != nil {
			return err
		}
		cli, err := client.NewClient(configuration.ServerAddr, configuration.ConfigFolder, configuration.TokenFilename)
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
