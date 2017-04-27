package sessions

import (
	"errors"
	"net/http"
	"time"

	"github.com/adam-hanna/sessions/sessionerrs"
	"github.com/adam-hanna/sessions/store"
	"github.com/adam-hanna/sessions/user"
	"github.com/garyburd/redigo/redis"
)

const (
	// DefaultExpirationDuration sets the default session expiration time in seconds
	DefaultExpirationDuration = 3 * 24 * time.Hour // 3 days
	// DefaultCSRFHeaderKey is the default key that will be used for reading csrf strings from requests
	DefaultCSRFHeaderKey = "X-CSRF-Token"
)

// Service is an object that contains information about this user's session
type Service struct {
	store   store.ServiceInterface
	options Options
}

// Options defines the behavior of the session
type Options struct {
	ExpirationDuration time.Duration
	Key                []byte
	IsDevEnv           bool
	CSRFHeaderKey      string
}

func init() {

}

// New returns a new session service
func New(store store.ServiceInterface, sessionOptions Options) (*Service, error) {
	if len(sessionOptions.Key) == 0 {
		return &Service{}, errors.New("no session key")
	}
	setDefaultOptions(&sessionOptions)
	return &Service{
		store:   store,
		options: sessionOptions,
	}, nil
}

// IssueUserSession grants a new user session, writes that session info to the redis db \
// and writes the session and csrf cookies on the http.ResponseWriter.
//
// This method should be called when a user logs in, for example.
func (s *Service) IssueUserSession(userID string, w http.ResponseWriter) (*user.Session, *sessionerrs.Custom) {
	userSession, err := user.New(userID, s.options.ExpirationDuration)
	if err != nil {
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// grab a redis connection from the pool
	c := s.store.GetConnectionFromPool()
	defer c.Close()

	// store the session in the db
	_, err = c.Do("HMSET", userSession.ID, "UserID", userSession.UserID, "CSRF", userSession.CSRF, "ExpiresAt", userSession.ExpiresAt.Unix())
	if err != nil {
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// set the expiration time of the redis key
	_, err = c.Do("EXPIREAT", userSession.ID, userSession.ExpiresAt.Unix())
	if err != nil {
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// sign the session id
	// note: the csrf string is already base64 encoded
	base64SessionValBytes := s.signAndEncodeSessionID(userSession.ID)

	// set the cookies
	sessionCookie := http.Cookie{
		Name:     "session",
		Value:    string(base64SessionValBytes[:]),
		Expires:  userSession.ExpiresAt,
		Path:     "/",
		HttpOnly: true,
		Secure:   !s.options.IsDevEnv,
	}
	// note: csrf strings are set on the cookie, but are expected to be received in the header
	csrfCookie := http.Cookie{
		Name:     "csrf",
		Value:    userSession.CSRF,
		Expires:  userSession.ExpiresAt,
		Path:     "/",
		HttpOnly: false,
		Secure:   !s.options.IsDevEnv,
	}
	http.SetCookie(w, &sessionCookie)
	http.SetCookie(w, &csrfCookie)

	return userSession, nil
}

// ClearUserSession is used to remove the user session from the db and clear the cookies on the ResponseWriter.
//
// This method should be called when a user logs out, for example.
func (s *Service) ClearUserSession(userSession *user.Session, w http.ResponseWriter) *sessionerrs.Custom {
	aLongTimeAgo := time.Now().Add(-1000 * time.Hour)

	// grab a redis connection from the pool
	c := s.store.GetConnectionFromPool()
	defer c.Close()

	// set the expiration time of the redis key
	_, err := c.Do("EXPIREAT", userSession.ID, aLongTimeAgo.Unix())
	if err != nil {
		return &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// set the cookies
	sessionCookie := http.Cookie{
		Name:     "session",
		Value:    "",
		Expires:  aLongTimeAgo,
		Path:     "/",
		HttpOnly: true,
		Secure:   !s.options.IsDevEnv,
	}
	// note: csrf strings are set on the cookie, but are expected to be received in the header
	csrfCookie := http.Cookie{
		Name:     "csrf",
		Value:    "",
		Expires:  aLongTimeAgo,
		Path:     "/",
		HttpOnly: false,
		Secure:   !s.options.IsDevEnv,
	}
	http.SetCookie(w, &sessionCookie)
	http.SetCookie(w, &csrfCookie)

	return nil
}

// GetUserSession returns a user session from a request. This method only returns valid sessions. Therefore, \
// sessions that have expired, that fail hmac signature verification, or that don't have matching csrf strings \
// will return a custom session error with code 401
func (s *Service) GetUserSession(r *http.Request) (*user.Session, *sessionerrs.Custom) {
	// read the session cookie
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, &sessionerrs.Custom{
				Code: 401,
				Err:  errors.New("no session cookie"),
			}
		}
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// read the csrf string
	csrfInReq := r.Header.Get(s.options.CSRFHeaderKey)
	if csrfInReq == "" {
		return nil, &sessionerrs.Custom{
			Code: 401,
			Err:  errors.New("no csrf string"),
		}
	}

	// decode the session cookie val
	sessionCookieValueBytes := []byte(sessionCookie.Value)
	decodedSessionCookieValueBytes, err := decode(sessionCookieValueBytes)
	if err != nil {
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// note: session uuid's are always 36 bytes long
	sessionIDBytes := decodedSessionCookieValueBytes[:36]
	sessionID := string(sessionIDBytes[:])
	hmacBytes := decodedSessionCookieValueBytes[36:]

	// verify the hmac signature
	verified := verifyHMAC(&sessionIDBytes, &hmacBytes, &s.options.Key)
	if !verified {
		return nil, &sessionerrs.Custom{
			Code: 401,
			Err:  errors.New("invalid session signature"),
		}
	}

	// check the session id in the database
	// first, grab a redis connection from the pool
	c := s.store.GetConnectionFromPool()
	defer c.Close()

	// check if the key exists
	exists, err := redis.Bool(c.Do("EXISTS", sessionID))
	if err != nil {
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}
	if !exists {
		return nil, &sessionerrs.Custom{
			Code: 401,
			Err:  errors.New("session is expired"),
		}
	}

	var userID string
	var csrf string
	var expiresAtSeconds int64
	reply, err := redis.Values(c.Do("HMGET", sessionID, "UserID", "CSRF", "ExpiresAt"))
	if err != nil {
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}
	if len(reply) < 3 {
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  errors.New("error retrieving session data from store"),
		}
	}
	if _, err := redis.Scan(reply, &userID, &csrf, &expiresAtSeconds); err != nil {
		return nil, &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// check that the csrf from the req matches the csrf in the session
	if csrf != csrfInReq {
		return nil, &sessionerrs.Custom{
			Code: 401,
			Err:  errors.New("csrf doesn't match session"),
		}
	}

	userSession := &user.Session{
		ID:        sessionID,
		UserID:    userID,
		CSRF:      csrf,
		ExpiresAt: time.Unix(expiresAtSeconds, 0),
	}

	return userSession, nil
}

// RefreshUserSession extends the ExpiresAt of a session by the Options.ExpirationDuration
func (s *Service) RefreshUserSession(userSession *user.Session, w http.ResponseWriter) *sessionerrs.Custom {
	newExpiresAt := time.Now().Add(s.options.ExpirationDuration).UTC()

	// update the provided user session
	userSession.ExpiresAt = newExpiresAt

	// grab a redis connection from the pool
	c := s.store.GetConnectionFromPool()
	defer c.Close()

	// extend the key in the db
	_, err := c.Do("EXPIREAT", userSession.ID, newExpiresAt.Unix())
	if err != nil {
		return &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}
	// update the field value
	_, err = c.Do("HSET", userSession.ID, "ExpiresAt", newExpiresAt.Unix())
	if err != nil {
		return &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// finally, update the cookies
	// note: the csrf string is already base64 encoded
	base64SessionValBytes := s.signAndEncodeSessionID(userSession.ID)

	// set the cookies
	sessionCookie := http.Cookie{
		Name:     "session",
		Value:    string(base64SessionValBytes[:]),
		Expires:  userSession.ExpiresAt,
		Path:     "/",
		HttpOnly: true,
		Secure:   !s.options.IsDevEnv,
	}
	// note: csrf strings are set on the cookie, but are expected to be received in the header
	csrfCookie := http.Cookie{
		Name:     "csrf",
		Value:    userSession.CSRF,
		Expires:  userSession.ExpiresAt,
		Path:     "/",
		HttpOnly: false,
		Secure:   !s.options.IsDevEnv,
	}
	http.SetCookie(w, &sessionCookie)
	http.SetCookie(w, &csrfCookie)

	return nil
}
