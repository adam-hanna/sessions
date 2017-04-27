package store

import (
	"github.com/garyburd/redigo/redis"
)

// ServiceInterface defines the behavior of the Redis struct
type ServiceInterface interface {
	GetConnectionFromPool() redis.Conn
}
