package user

import (
	"time"

	"github.com/pborman/uuid"
)

// Session is a user's session struct
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	JSON      string
}

// New returns a new user Session
func New(userID string, json string, duration time.Duration) *Session {
	return &Session{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(duration).UTC(),
		JSON:      json,
	}
}
