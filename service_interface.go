package sessions

import (
	"net/http"

	"github.com/adam-hanna/sessions/user"
)

// ServiceInterface defines the methods performed by the session service
type ServiceInterface interface {
	IssueUserSession(userID string, json string, w http.ResponseWriter) (*user.Session, error)
	ClearUserSession(userSession *user.Session, w http.ResponseWriter) error
	GetUserSession(r *http.Request) (*user.Session, error)
	ExtendUserSession(userSession *user.Session, r *http.Request, w http.ResponseWriter) error
}
