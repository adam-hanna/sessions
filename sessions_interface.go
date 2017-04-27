package sessions

import (
	"net/http"

	"github.com/adam-hanna/sessions/sessionerrs"
	"github.com/adam-hanna/sessions/user"
)

// ServiceInterface is used to track state in an http application
type ServiceInterface interface {
	IssueUserSession(userID string, w http.ResponseWriter) (*user.Session, *sessionerrs.Custom)
	ClearUserSession(userSession *user.Session, w http.ResponseWriter) *sessionerrs.Custom
	GetUserSession(r *http.Request) (*user.Session, *sessionerrs.Custom)
	RefreshUserSession(userSession *user.Session, w http.ResponseWriter) *sessionerrs.Custom
}
