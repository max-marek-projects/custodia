package main

import "github.com/spf13/cobra"

// rootCmd implements root command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update secret item on server",
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
