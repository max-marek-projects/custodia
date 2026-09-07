package client

import "time"

type Session struct {
	Password  []byte
	ExpiresAt time.Time
}

func (s *Session) LoggedIn() bool {
	if len(s.Password) == 0 {
		return false
	}
	if time.Now().After(s.ExpiresAt) {
		s.Clear()
		return false
	}
	return true
}

func (s *Session) Clear() {
	s.Password = nil
	s.ExpiresAt = time.Time{}
}
