// Package client provides an interactive REPL (Read-Eval-Print Loop) for the
// Custodia CLI. It uses readline for line editing and history, and integrates
// with Cobra commands for execution.
package client

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"
	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
)

// RunREPL starts an interactive shell that reads commands from the user,
// parses them into arguments, and executes them using the provided root Cobra
// command.
//
// The REPL supports:
//   - Line editing with history (saved to ~/.custodia_history)
//   - Shell-like argument parsing (quotes, escapes, etc.)
//   - Built-in commands: help, exit, quit
//   - Any command registered with the root command
//
// Parameters:
//   - rootCmd: the root Cobra command to execute. Its subcommands and flags
//     will be available in the REPL.
//
// Returns:
//   - error: nil if the REPL exits normally (via exit/quit or EOF),
//     or an error if readline initialization fails.
func RunREPL(rootCmd *cobra.Command) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "custodia> ",
		HistoryFile:     "~/.custodia_history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("failed to create readline instance: %w", err)
	}
	defer rl.Close()

	fmt.Println("Custodia interactive shell")
	fmt.Println("Type 'help' for help.")
	fmt.Println("Type 'exit' to quit.")
	fmt.Println()

	for {
		line, err := rl.Readline()
		if err != nil {
			// EOF (Ctrl+D) or interrupt (Ctrl+C) – exit gracefully
			return nil
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}

		args, err := shellwords.Parse(line)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			fmt.Println("Error:", err)
		}
		fmt.Println()
	}
}
