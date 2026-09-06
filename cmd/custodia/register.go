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

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "register to custodia service",
	RunE: func(cmd *cobra.Command, args []string) error {
		configuration, err := config.NewClientConf()
		if err != nil {
			return err
		}
		err = logger.Initialize(configuration.LoggerLevel)
		if err != nil {
			return err
		}
		login, err := cmd.Flags().GetString("login")
		if err != nil {
			return err
		}
		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		}

		if login == "" {
			fmt.Print("Please type your login: ")
			fmt.Scanln(&login)
		}
		if password == "" {
			fmt.Print("Please type your password: ")
			bytePass, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return err
			}
			password = string(bytePass)
			fmt.Println()
		}
		cli, err := client.NewClient(configuration.ServerAddr, configuration.ConfigFolder, configuration.TokenFilename)
		if err != nil {
			return err
		}
		defer cli.Close()
		if err := cli.Register(context.Background(), login, password); err != nil {
			return err
		}
		fmt.Println("registered successfully")
		return nil
	},
}

func init() {
	registerCmd.Flags().StringP("login", "l", "", "your login for custodia service")
	registerCmd.Flags().StringP("password", "p", "", "your password for custodia service")
	rootCmd.AddCommand(registerCmd)
}
