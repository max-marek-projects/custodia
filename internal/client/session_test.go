package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSession_LoggedIn(t *testing.T) {
	tests := []struct {
		name      string
		password  []byte
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "valid session",
			password:  []byte("secret"),
			expiresAt: time.Now().Add(time.Hour),
			want:      true,
		},
		{
			name:      "empty password",
			password:  nil,
			expiresAt: time.Now().Add(time.Hour),
			want:      false,
		},
		{
			name:      "expired session",
			password:  []byte("secret"),
			expiresAt: time.Now().Add(-time.Hour),
			want:      false,
		},
		{
			name:      "expired at exact now (should be false)",
			password:  []byte("secret"),
			expiresAt: time.Now(),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{
				Password:  tt.password,
				ExpiresAt: tt.expiresAt,
			}
			got := s.LoggedIn()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSession_LoggedIn_ExpiredClears(t *testing.T) {
	s := &Session{
		Password:  []byte("secret"),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	assert.False(t, s.LoggedIn())
	assert.Nil(t, s.Password)
	assert.True(t, s.ExpiresAt.IsZero())
}

func TestSession_LoggedIn_ValidDoesNotClear(t *testing.T) {
	s := &Session{
		Password:  []byte("secret"),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	assert.True(t, s.LoggedIn())
	assert.NotNil(t, s.Password)
	assert.False(t, s.ExpiresAt.IsZero())
}

func TestSession_Clear(t *testing.T) {
	s := &Session{
		Password:  []byte("secret"),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	s.Clear()
	assert.Nil(t, s.Password)
	assert.True(t, s.ExpiresAt.IsZero())
}

func TestSession_LoggedIn_WithZeroTime(t *testing.T) {
	s := &Session{
		Password:  []byte("secret"),
		ExpiresAt: time.Time{}, // zero time
	}
	// zero time is before now, so it will be expired
	assert.False(t, s.LoggedIn())
	assert.Nil(t, s.Password)
}
