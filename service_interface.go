package sessions

import (
	"net/http"

	"github.com/adam-hanna/sessions/sessionerrs"
	"github.com/adam-hanna/sessions/user"
)

// ServiceInterface defines the methods performed by the session service
type ServiceInterface interface {
	IssueUserSession(userID string, json string, w http.ResponseWriter) (*user.Session, *sessionerrs.Custom)
	ClearUserSession(userSession *user.Session, w http.ResponseWriter) *sessionerrs.Custom
	GetUserSession(r *http.Request) (*user.Session, *sessionerrs.Custom)
	ExtendUserSession(userSession *user.Session, r *http.Request, w http.ResponseWriter) *sessionerrs.Custom
}
