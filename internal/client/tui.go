package client

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"
	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
)

func RunREPL(rootCmd *cobra.Command) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "custodia> ",
		HistoryFile:     "~/.custodia_history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()
	fmt.Println("Custodia interactive shell")
	fmt.Println("Type 'help' for help.")
	fmt.Println("Type 'exit' to quit.")
	fmt.Println()
	for {
		line, err := rl.Readline()
		if err != nil {
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
