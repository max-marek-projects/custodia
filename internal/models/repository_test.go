// models/card_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCardData(t *testing.T) {
	t.Run("valid card", func(t *testing.T) {
		card, err := NewCardData("1234 5600 0000 0009", "John Doe", "12/2026", "123")
		require.NoError(t, err)
		assert.NotNil(t, card)
		assert.Equal(t, "1234 5600 0000 0009", card.Number)
		assert.Equal(t, "John Doe", card.HolderName)
		assert.Equal(t, "12/2026", card.ExpirationDate)
		assert.Equal(t, "123", card.CVV)
	})

	t.Run("invalid number", func(t *testing.T) {
		_, err := NewCardData("invalid", "John Doe", "12/2026", "123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid card number")
	})

	t.Run("invalid holder", func(t *testing.T) {
		_, err := NewCardData("1234 5600 0000 0009", "", "12/2026", "123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid card holder name")
	})

	t.Run("invalid expiration", func(t *testing.T) {
		_, err := NewCardData("1234 5600 0000 0009", "John Doe", "13/2026", "123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid expiration date")
	})

	t.Run("invalid CVV", func(t *testing.T) {
		_, err := NewCardData("1234 5600 0000 0009", "John Doe", "12/2026", "12")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid CVV")
	})
}
