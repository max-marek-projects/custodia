package client

import (
	"io"
	"os"
	"testing"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShellwordsParsing tests the shellwords parsing logic.
func TestShellwordsParsing(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    []string
		wantErr bool
	}{
		{"simple", "hello world", []string{"hello", "world"}, false},
		{"quoted", `echo "hello world"`, []string{"echo", "hello world"}, false},
		{"escaped", `echo hello\ world`, []string{"echo", "hello world"}, false},
		{"unclosed quote", `echo "hello`, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := shellwords.Parse(tt.line)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, args)
			}
		})
	}
}

// TestCommandExecution tests that commands are executed correctly.
func TestCommandExecution(t *testing.T) {
	executed := false
	rootCmd := &cobra.Command{Use: "root"}
	testCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			executed = true
			return nil
		},
	}
	rootCmd.AddCommand(testCmd)

	rootCmd.SetArgs([]string{"test"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.True(t, executed)
}

// TestCommandExecutionWithError tests error handling in command execution.
func TestCommandExecutionWithError(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	failCmd := &cobra.Command{
		Use:  "fail",
		RunE: func(cmd *cobra.Command, args []string) error { return assert.AnError },
	}
	rootCmd.AddCommand(failCmd)

	rootCmd.SetArgs([]string{"fail"})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

// TestREPLIntegration runs the full REPL loop with real stdin/stdout pipes.
func TestREPLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a pipe for stdin
	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	defer stdinR.Close()
	defer stdinW.Close()

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = stdinR

	// Create a pipe for stdout
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)
	defer stdoutR.Close()
	defer stdoutW.Close()

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = stdoutW

	// Create a root command with a simple subcommand
	rootCmd := &cobra.Command{Use: "root"}
	helloCmd := &cobra.Command{
		Use: "hello",
		Run: func(cmd *cobra.Command, args []string) { /* do nothing */ },
	}
	rootCmd.AddCommand(helloCmd)

	// Write commands to stdin
	go func() {
		defer stdinW.Close()
		stdinW.WriteString("hello\n")
		stdinW.WriteString("exit\n")
	}()

	// Run the REPL
	err = RunREPL(rootCmd)
	require.NoError(t, err)

	// Read stdout
	stdoutW.Close()
	out, err := io.ReadAll(stdoutR)
	require.NoError(t, err)

	output := string(out)
	assert.Contains(t, output, "Custodia interactive shell")
	assert.Contains(t, output, "Type 'help' for help.")
	assert.Contains(t, output, "Type 'exit' to quit.")
}
