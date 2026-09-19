// Package client provides session management for authenticated CLI sessions.
package client

import "time"

// Session represents an authenticated user session with a password (or token)
// and an expiration time. It provides methods to check login status and clear
// the session data.
type Session struct {
	// Password stores the user's password (or hashed token) for the session.
	Password []byte
	// ExpiresAt is the time after which the session is considered expired.
	ExpiresAt time.Time
}

// LoggedIn checks whether the session is currently valid.
// A session is valid if:
//   - Password is not empty
//   - ExpiresAt is in the future (strictly greater than now)
//
// If the session has expired or equals now, it calls Clear() and returns false.
//
// Returns:
//   - bool: true if the session is valid, false otherwise.
func (s *Session) LoggedIn() bool {
	if len(s.Password) == 0 {
		return false
	}
	// Expired if now is not before ExpiresAt (i.e., now >= ExpiresAt)
	if !time.Now().Before(s.ExpiresAt) {
		s.Clear()
		return false
	}
	return true
}

// Clear resets the session by clearing the password and resetting
// the expiration time to the zero value. This should be called when
// logging out or when the session expires.
func (s *Session) Clear() {
	s.Password = nil
	s.ExpiresAt = time.Time{}
}
