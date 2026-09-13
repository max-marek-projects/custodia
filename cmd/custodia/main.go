// Package main implements a CLI application for the Custodia password manager.
// It provides both interactive REPL mode (when no arguments are given) and
// command‑line subcommand mode (when arguments are provided).
//
// The root command and global session are defined here and shared across all
// subcommands (login, logout, create, get, update, delete, rollback, refresh).
package main

import (
	"fmt"
	"os"

	"github.com/max-marek-projects/custodia/internal/client"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/spf13/cobra"
)

// rootCmd is the root command of the Custodia CLI.
// It is used both in command-line mode and as the command registry for the REPL.
// The SilenceUsage flag hides the usage help on command errors.
var rootCmd = &cobra.Command{
	Use:          "custodia",
	Short:        "Custodia - simple but secure password manager",
	SilenceUsage: true, // Prevents automatic usage printing on errors.
}

// session holds the user's session state (password and expiration time).
// It is shared globally across all subcommands and is used by the client
// to determine if the user is logged in and to provide authentication tokens.
var session = &client.Session{}

// global variables that can be rewritten by flags
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func init() {
	rootCmd.Version = utils.OrNA(buildVersion)
	rootCmd.SetVersionTemplate(fmt.Sprintf("Custodia version {{.Version}}\nBuilt on %s\n", buildDate))
}

// run is the entry point of the application.
// It determines the execution mode:
//   - If command-line arguments are provided, it executes the root command.
//   - If no arguments are provided, it starts an interactive REPL.
//
// Any error during execution is printed to stderr and the process exits with code 1.
func run() error {
	fmt.Println("Build version:", utils.OrNA(buildVersion))
	fmt.Println("Build date:", utils.OrNA(buildDate))
	fmt.Println("Build commit:", utils.OrNA(buildCommit))
	if len(os.Args) > 1 {
		// Command-line mode: execute the requested subcommand.
		if err := rootCmd.Execute(); err != nil {
			fmt.Println(err)
			return err
		}
		return nil
	}
	// REPL mode: interactive shell for all commands.
	if err := client.RunREPL(rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
