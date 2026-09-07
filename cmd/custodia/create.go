package main

import "github.com/spf13/cobra"

// rootCmd implements root command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create new secret item on server",
}

func init() {
	rootCmd.AddCommand(createCmd)
}
