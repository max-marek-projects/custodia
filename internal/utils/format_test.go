package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrNA(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "N/A",
		},
		{
			name:  "non-empty string",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "string with spaces",
			input: "  ",
			want:  "  ", // current behavior: spaces are not considered empty
		},
		{
			name:  "string with only newline",
			input: "\n",
			want:  "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OrNA(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
