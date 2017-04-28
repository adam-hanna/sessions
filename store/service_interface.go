package store

import (
	"github.com/adam-hanna/sessions/sessionerrs"
	"github.com/adam-hanna/sessions/user"
)

// ServiceInterface defines the behavior of the session store
type ServiceInterface interface {
	SaveUserSession(userSession *user.Session) *sessionerrs.Custom
	DeleteUserSession(sessionID string) *sessionerrs.Custom
	FetchValidUserSession(sessionID string) (*user.Session, *sessionerrs.Custom)
}
