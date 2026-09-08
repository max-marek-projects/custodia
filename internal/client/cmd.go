// Package client provides helper functions for reading user input from the command line,
// including flags, interactive prompts, and secure password input.
package client

import (
	"encoding/json"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// ReadStringValue reads a string value from a command flag or interactively from stdin.
// If the flag is provided and non-empty, that value is returned.
// Otherwise, it prompts the user to type the value and reads from stdin.
//
// Parameters:
//   - cmd: the Cobra command containing the flag.
//   - flagName: the name of the flag to check.
//
// Returns:
//   - string: the value read (either from flag or from stdin).
//   - error: non-nil if reading the flag fails or scanning stdin fails.
func ReadStringValue(cmd *cobra.Command, flagName string) (string, error) {
	value, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return "", fmt.Errorf("failed to get value from flag `%s`: %w", flagName, err)
	}
	if value != "" {
		return value, nil
	}
	fmt.Printf("Please type %s: ", flagName)
	_, err = fmt.Scanln(&value)
	if err != nil {
		return "", fmt.Errorf("failed to get %s from command line: %w", flagName, err)
	}
	return value, nil
}

// ReadMapValue reads a map[string]string from a command flag or interactively from stdin.
// The expected input is a JSON object (e.g., `{"key":"value"}`).
// If the flag is provided and non-empty, it is parsed as JSON.
// Otherwise, it prompts the user to type the JSON and reads from stdin.
//
// Parameters:
//   - cmd: the Cobra command containing the flag.
//   - flagName: the name of the flag to check.
//
// Returns:
//   - map[string]string: the parsed metadata.
//   - error: non-nil if reading the flag fails, scanning stdin fails, or JSON parsing fails.
func ReadMapValue(cmd *cobra.Command, flagName string) (map[string]string, error) {
	rawValue, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata from flag: %w", err)
	}
	if rawValue == "" {
		fmt.Printf("Please type %s: ", flagName)
		_, err = fmt.Scanln(&rawValue)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s from command line: %w", flagName, err)
		}
	}
	var metadata map[string]string
	err = json.Unmarshal([]byte(rawValue), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	return metadata, nil
}

// ReadSecret reads a secret (e.g., password) from the terminal without echoing the input.
// It uses term.ReadPassword to hide the typed characters.
//
// Parameters:
//   - name: a descriptive name for the secret (used in the prompt).
//
// Returns:
//   - []byte: the secret as a byte slice.
//   - error: non-nil if reading from the terminal fails.
func ReadSecret(name string) ([]byte, error) {
	fmt.Printf("Please type %s: ", name)
	secretData, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", name, err)
	}
	fmt.Println()
	return secretData, nil
}
