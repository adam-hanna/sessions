package transport

import (
	"net/http"

	"github.com/adam-hanna/sessions/sessionerrs"
	"github.com/adam-hanna/sessions/user"
)

// ServiceInterface defines the methods performed by the transport service
type ServiceInterface interface {
	SetSessionOnResponse(session string, userSession *user.Session, w http.ResponseWriter) *sessionerrs.Custom
	DeleteSessionFromResponse(w http.ResponseWriter) *sessionerrs.Custom
	FetchSessionIDFromRequest(r *http.Request) (string, *sessionerrs.Custom)
}
