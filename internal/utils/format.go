// Package utils provides helper functions for variables formatting.
package utils

// OrNA replaces empty string value with N/A
func OrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
