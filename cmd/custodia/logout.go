package main

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/config"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "logout from custodia service on current device",
	RunE: func(cmd *cobra.Command, args []string) error {
		configuration, err := config.NewClientConf()
		if err != nil {
			return fmt.Errorf("failed to initialize configuration: %w", err)
		}
		err = logger.Initialize(configuration.LoggerLevel)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		allDevices, err := cmd.Flags().GetBool("all")
		if err != nil {
			return fmt.Errorf("failed to get `all` value from flag: %w", err)
		}
		cli, err := client.NewClient(configuration.ServerAddr, configuration.ConfigFolder, configuration.TokenFilename)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cli.Close()
		if err := cli.Logout(context.Background(), allDevices); err != nil {
			return fmt.Errorf("failed to logout: %w", err)
		}
		fmt.Println("logged out successfully")
		return nil
	},
}

func init() {
	logoutCmd.Flags().BoolP("all", "a", false, "logout from all devices")
	rootCmd.AddCommand(logoutCmd)
}
