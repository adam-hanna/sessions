package store

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	// Opts is used to store the behavior of the service
	Opts Options
)

const (
	// DefaultConnectionAddress defines the default connection address of the redis server
	DefaultConnectionAddress = ":6379"
	// DefaultMaxIdleConnections sets the maximum number of idle connections on the redis server
	DefaultMaxIdleConnections = 3
	// DefaultMaxActiveConnections sets the maximum number of active connections on the redis server
	DefaultMaxActiveConnections = 10
	// DefaultIdleTimeoutDuration sets the maximum duration to wait before closing an idle connection on the redis server
	DefaultIdleTimeoutDuration = 10 * time.Second
)

// Service is a session store backed by a redis db
type Service struct {
	pool *redis.Pool
}

// Options dictates the behavior of the session store
type Options struct {
	ConnectionAddress    string
	MaxIdleConnections   int
	MaxActiveConnections int
	IdleTimeoutDuration  time.Duration
}

func init() {

}

// New returns a new session store connected to a redis db
func New(opts Options) *Service {
	setDefaultOptions(&opts)
	return &Service{
		pool: &redis.Pool{
			MaxActive:   opts.MaxActiveConnections,
			MaxIdle:     opts.MaxIdleConnections,
			IdleTimeout: opts.IdleTimeoutDuration,
			Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", opts.ConnectionAddress) },
		},
	}
}

// GetConnectionFromPool creates a new session from a pool
func (s *Service) GetConnectionFromPool() redis.Conn {
	return s.pool.Get()
}
