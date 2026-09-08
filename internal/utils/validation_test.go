package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCardNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid with spaces", "1234 5600 0000 0009", false},
		{"valid without spaces", "1234560000000009", false},
		{"valid with dashes", "1234-5600-0000-0009", false},
		{"too short", "4532 1488 0343 646", true},
		{"too long", "4532 1488 0343 64670", true},
		{"contains letters", "4532 1488 0343 646A", true},
		{"fails Luhn", "1111 1111 1111 1111", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCardNumber(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCVV(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"123", false},
		{"000", false},
		{"12", true},
		{"1234", true},
		{"12a", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateCVV(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateExpirationDate(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"12/2025", false},
		{"01/2030", false},
		{"13/2025", true},
		{"00/2025", true},
		{"12/25", true},
		{"12-2025", true},
		{"12/20250", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateExpirationDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCardHolder(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"John Doe", false},
		{"Jean-Luc", false},
		{"Mr. Smith", false},
		{"J", false},
		{"", true},
		{"John123", true},
		{"John@Doe", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateCardHolder(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
