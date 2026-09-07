package main

import "github.com/spf13/cobra"

// rootCmd implements root command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get secret item from server",
}

func init() {
	rootCmd.AddCommand(getCmd)
}
