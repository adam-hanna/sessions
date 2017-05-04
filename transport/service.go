package transport

import (
	"errors"
	"net/http"
	"time"

	"github.com/adam-hanna/sessions/user"
)

const (
	// DefaultCookieName is the default cookie name used
	DefaultCookieName = "session"
	// DefaultCookiePath is the default cookie path
	DefaultCookiePath = "/"
	// DefaultHTTPOnlyCookie is the default HTTPOnly option of the cookie
	// DefaultHTTPOnlyCookie = true // changing this to false, the uninitialized val, for now
	// DefaultSecureCookie is the default Secure option of the cookie
	// DefaultSecureCookie = true // changing this to false, the uninitialized val, for now
)

// ErrNoSessionOnRequest is thrown when a session is not found on a request
var ErrNoSessionOnRequest = errors.New("no session on request")

// Service writes sessions on responseWriters and reads sessions from requests
type Service struct {
	options Options
}

// Options defines the behavior of the transport service
type Options struct {
	CookieName string
	CookiePath string
	HTTPOnly   bool
	Secure     bool
}

// New returns a new transport service
func New(options Options) *Service {
	setDefaultOptions(&options)
	return &Service{
		options: options,
	}
}

// SetSessionOnResponse sets a signed session id and a user session on a responseWriter
func (s *Service) SetSessionOnResponse(signedSessionID string, userSession *user.Session, w http.ResponseWriter) error {
	sessionCookie := http.Cookie{
		Name:     s.options.CookieName,
		Value:    signedSessionID,
		Expires:  userSession.ExpiresAt,
		Path:     s.options.CookiePath,
		HttpOnly: s.options.HTTPOnly,
		Secure:   s.options.Secure,
	}
	http.SetCookie(w, &sessionCookie)

	return nil
}

// DeleteSessionFromResponse deletes a user session from a responseWriter
func (s *Service) DeleteSessionFromResponse(w http.ResponseWriter) error {
	aLongTimeAgo := time.Now().Add(-1000 * time.Hour)
	nullSessionCookie := http.Cookie{
		Name:     s.options.CookieName,
		Value:    "",
		Expires:  aLongTimeAgo,
		Path:     s.options.CookiePath,
		HttpOnly: s.options.HTTPOnly,
		Secure:   s.options.Secure,
	}
	http.SetCookie(w, &nullSessionCookie)

	return nil
}

// FetchSessionIDFromRequest retrieves a signed session id from a request
func (s *Service) FetchSessionIDFromRequest(r *http.Request) (string, error) {
	sessionCookie, err := r.Cookie(s.options.CookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", ErrNoSessionOnRequest
		}
		return "", err
	}

	return sessionCookie.Value, nil
}
