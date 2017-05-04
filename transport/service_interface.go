package transport

import (
	"net/http"

	"github.com/adam-hanna/sessions/user"
)

// ServiceInterface defines the methods performed by the transport service
type ServiceInterface interface {
	SetSessionOnResponse(session string, userSession *user.Session, w http.ResponseWriter) error
	DeleteSessionFromResponse(w http.ResponseWriter) error
	FetchSessionIDFromRequest(r *http.Request) (string, error)
}
