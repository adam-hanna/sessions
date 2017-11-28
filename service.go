package sessions

import (
	"net/http"
	"time"

	"github.com/adam-hanna/sessions/auth"
	"github.com/adam-hanna/sessions/store"
	"github.com/adam-hanna/sessions/transport"
	"github.com/adam-hanna/sessions/user"
)

const (
	// DefaultExpirationDuration sets the default session expiration duration
	DefaultExpirationDuration = 3 * 24 * time.Hour // 3 days
)

// Service provides session service for http servers
type Service struct {
	store     store.ServiceInterface
	auth      auth.ServiceInterface
	transport transport.ServiceInterface
	options   Options
}

// Options defines the behavior of the session service
type Options struct {
	ExpirationDuration time.Duration
}

// New returns a new session service
func New(store store.ServiceInterface, auth auth.ServiceInterface, transport transport.ServiceInterface, options Options) *Service {
	setDefaultOptions(&options)
	return &Service{
		store:     store,
		auth:      auth,
		transport: transport,
		options:   options,
	}
}

// IssueUserSession grants a new user session, writes that session info to the store \
// and writes the session on the http.ResponseWriter.
//
// This method should be called when a user logs in, for example.
func (s *Service) IssueUserSession(userID string, json string, w http.ResponseWriter) (*user.Session, error) {
	userSession := user.New(userID, json, s.options.ExpirationDuration)

	// sign the session id
	signedSessionID, err := s.auth.SignAndBase64Encode(userSession.ID)
	if err != nil {
		return nil, err
	}

	// save the session in the store
	if err = s.store.SaveUserSession(userSession); err != nil {
		return nil, err
	}

	// set the session on the responseWriter
	return userSession, s.transport.SetSessionOnResponse(signedSessionID, userSession, w)
}

// ClearUserSession is used to remove the user session from the store and clear the cookies on the ResponseWriter.
//
// This method should be called when a user logs out, for example.
func (s *Service) ClearUserSession(userSession *user.Session, w http.ResponseWriter) error {
	// delete the session from the store
	if err := s.store.DeleteUserSession(userSession.ID); err != nil {
		return err
	}

	// delete the session from the response
	return s.transport.DeleteSessionFromResponse(w)
}

// GetUserSession returns a user session from a request. This method only returns valid sessions. Therefore, \
// sessions that have expired, or that fail signature verification will return a nil pointer to a user.Session
func (s *Service) GetUserSession(r *http.Request) (*user.Session, error) {
	// read the session from the request
	signedSessionID, err := s.transport.FetchSessionIDFromRequest(r)
	if err != nil {
		if err == transport.ErrNoSessionOnRequest {
			// note a nil user.Session pointer indicates a 401 unauthorized
			return nil, nil
		}

		return nil, err
	}

	// decode the signedSessionID
	sessionID, err := s.auth.VerifyAndDecode(signedSessionID)
	if err != nil {
		if err == auth.ErrInvalidSession {
			return nil, nil
		}

		return nil, err
	}

	// try fetching a valid session from the store
	return s.store.FetchValidUserSession(sessionID)
}

// ExtendUserSession extends the ExpiresAt of a session by the Options.ExpirationDuration
//
// Note that this function must be called, manually! Extension of user session expiry's does not happen automatically!
func (s *Service) ExtendUserSession(userSession *user.Session, r *http.Request, w http.ResponseWriter) error {
	newExpiresAt := time.Now().Add(s.options.ExpirationDuration).UTC()

	// update the provided user session
	userSession.ExpiresAt = newExpiresAt

	// save the session in the store with the extended expiry
	if err := s.store.SaveUserSession(userSession); err != nil {
		return err
	}

	// fetch the signed session id from the request
	signedSessionID, err := s.transport.FetchSessionIDFromRequest(r)
	if err != nil {
		return err
	}

	// finally, set the session on the responseWriter
	return s.transport.SetSessionOnResponse(signedSessionID, userSession, w)
}
