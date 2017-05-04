package store

import (
	"errors"
	"time"

	"github.com/adam-hanna/sessions/user"
	"github.com/garyburd/redigo/redis"
)

const (
	// DefaultConnectionAddress defines the default connection address of the redis server
	DefaultConnectionAddress = ":6379"
	// DefaultMaxIdleConnections sets the maximum number of idle connections on the redis server
	DefaultMaxIdleConnections = 3
	// DefaultIdleTimeoutDuration sets the maximum duration to wait before closing an idle connection on the redis server
	DefaultIdleTimeoutDuration = 10 * time.Second
	// DefaultMaxActiveConnections sets the maximum number of active connections on the redis server
	// DefaultMaxActiveConnections = 10 // changing this to 0, the uninitialized val, for now
)

var (
	// ErrRetrievingSession is thrown if there was an error, other than an invalid session, retrieving the \
	// session from the store
	ErrRetrievingSession = errors.New("error retrieving session data from store")
)

// Service is a session store backed by a redis db
type Service struct {
	// Pool is a redigo *redis.Pool
	Pool *redis.Pool
}

// Options defines the behavior of the session store
type Options struct {
	ConnectionAddress    string
	MaxIdleConnections   int
	MaxActiveConnections int
	IdleTimeoutDuration  time.Duration
}

// New returns a new session store connected to a redis db
// Alternatively, you can build your own redis store with &Service{Pool: yourCustomPool,}
func New(options Options) *Service {
	setDefaultOptions(&options)
	return &Service{
		Pool: &redis.Pool{
			MaxActive:   options.MaxActiveConnections,
			MaxIdle:     options.MaxIdleConnections,
			IdleTimeout: options.IdleTimeoutDuration,
			Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", options.ConnectionAddress) },
		},
	}
}

// SaveUserSession saves a user session in the store
func (s *Service) SaveUserSession(userSession *user.Session) error {
	c := s.Pool.Get()
	defer c.Close()

	// note @adam-hanna: should I pipeline these requests?
	if _, err := c.Do("HMSET", userSession.ID, "UserID", userSession.UserID, "JSON", userSession.JSON, "ExpiresAtSeconds", userSession.ExpiresAt.Unix()); err != nil {
		return err
	}

	// set the expiration time of the redis key
	if _, err := c.Do("EXPIREAT", userSession.ID, userSession.ExpiresAt.Unix()); err != nil {
		return err
	}

	return nil
}

// DeleteUserSession deletes a user session from the store
func (s *Service) DeleteUserSession(sessionID string) error {
	// grab a redis connection from the pool
	c := s.Pool.Get()
	defer c.Close()

	// set the expiration time of the redis key
	aLongTimeAgo := time.Now().Add(-1000 * time.Hour)
	if _, err := c.Do("EXPIREAT", sessionID, aLongTimeAgo.Unix()); err != nil {
		return err
	}

	return nil
}

// FetchValidUserSession returns a valid user session or an err if the session has expired or does not exist. \
// If a valid session does not exist, this function should return a nil pointer
func (s *Service) FetchValidUserSession(sessionID string) (*user.Session, error) {
	// check the session id in the database
	// first, grab a redis connection from the pool
	c := s.Pool.Get()
	defer c.Close()

	// note @adam-hanna: should I pipeline these requests?
	// check if the key exists
	exists, err := redis.Bool(c.Do("EXISTS", sessionID))
	if err != nil {
		return nil, err
	}
	// note: if a valid session does not exist, this function should return a nil pointer
	if !exists {
		return nil, nil
	}

	var userID string
	var json string
	var expiresAtSeconds int64
	reply, err := redis.Values(c.Do("HMGET", sessionID, "UserID", "JSON", "ExpiresAtSeconds"))
	if err != nil {
		return nil, err
	}
	if len(reply) < 3 {
		return nil, ErrRetrievingSession
	}
	for idx := range reply {
		if reply[idx] == nil {
			return nil, ErrRetrievingSession
		}
	}
	if _, err := redis.Scan(reply, &userID, &json, &expiresAtSeconds); err != nil {
		return nil, err
	}

	return &user.Session{
		ID:        sessionID,
		UserID:    userID,
		JSON:      json,
		ExpiresAt: time.Unix(expiresAtSeconds, 0),
	}, nil
}
