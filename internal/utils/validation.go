// Package utils provides helper functions for data validation.
package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateCardNumber checks that the card number matches the format "0000 0000 0000 0000"
// and passes the Luhn algorithm.
func ValidateCardNumber(number string) error {
	// Remove all spaces and dashes
	normalized := strings.ReplaceAll(strings.ReplaceAll(number, " ", ""), "-", "")
	// Must be exactly 16 digits
	if len(normalized) != 16 {
		return fmt.Errorf("card number must contain exactly 16 digits")
	}
	if !regexp.MustCompile(`^\d{16}$`).MatchString(normalized) {
		return fmt.Errorf("card number must contain only digits")
	}
	// Luhn algorithm check
	if !validateLuhnAlgorithm(normalized) {
		return fmt.Errorf("card number failed Luhn check")
	}
	return nil
}

// validateLuhnAlgorithm checks if a number string satisfies the Luhn algorithm (mod 10).
// Returns true for valid credit card‑style numbers, false otherwise.
func validateLuhnAlgorithm(number string) bool {
	number = strings.ReplaceAll(number, " ", "")
	if len(number) < 2 {
		return false
	}
	sum := 0
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		ch := number[i]
		if ch < '0' || ch > '9' {
			return false
		}
		digit := int(ch - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}

// ValidateCVV checks that the CVV is a string of exactly 3 digits.
func ValidateCVV(cvv string) error {
	if !regexp.MustCompile(`^\d{3}$`).MatchString(cvv) {
		return fmt.Errorf("CVV must be exactly 3 digits")
	}
	return nil
}

// ValidateExpirationDate checks that the expiration date has format "MM/YYYY".
func ValidateExpirationDate(date string) error {
	if !regexp.MustCompile(`^(0[1-9]|1[0-2])/\d{4}$`).MatchString(date) {
		return fmt.Errorf("expiration date must be in format MM/YYYY")
	}
	return nil
}

// ValidateCardHolder validates the card holder name (non‑empty, only letters, spaces, hyphens, dots).
func ValidateCardHolder(name string) error {
	if name == "" {
		return fmt.Errorf("card holder name cannot be empty")
	}
	if !regexp.MustCompile(`^[a-zA-Z\s\.\-]+$`).MatchString(name) {
		return fmt.Errorf("card holder name may contain only letters, spaces, hyphens, and dots")
	}
	return nil
}
