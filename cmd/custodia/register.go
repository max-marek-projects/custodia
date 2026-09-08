package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "register to custodia service",
	RunE: func(cmd *cobra.Command, args []string) error {
		login, err := cmd.Flags().GetString("login")
		if err != nil {
			return err
		}
		if login == "" {
			fmt.Print("Please type your login: ")
			fmt.Scanln(&login)
		}
		fmt.Print("Please type your password: ")
		session.Password, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()
		cli, configuration, err := client.NewClient(session)
		if err != nil {
			return err
		}
		defer cli.Close()
		if err := cli.Register(context.Background(), login, session.Password, configuration.SessionTTL.Duration()); err != nil {
			return err
		}
		fmt.Println("registered successfully")
		return nil
	},
}

func init() {
	registerCmd.Flags().StringP("login", "l", "", "your login for custodia service")
	rootCmd.AddCommand(registerCmd)
}
