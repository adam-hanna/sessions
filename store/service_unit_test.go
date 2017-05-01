// +build unit

package store

import (
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
)

var testOptions = Options{ConnectionAddress: "test", MaxIdleConnections: 5, MaxActiveConnections: 5, IdleTimeoutDuration: 1 * time.Second}

// TestNew tests the New function
func TestNew(t *testing.T) {
	var tests = []struct {
		input    Options
		expected Service
	}{
		{
			Options{},
			Service{
				Pool: &redis.Pool{
					MaxActive:   0,
					MaxIdle:     DefaultMaxIdleConnections,
					IdleTimeout: DefaultIdleTimeoutDuration,
					Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", DefaultConnectionAddress) },
				},
			},
		},
		{
			testOptions,
			Service{
				Pool: &redis.Pool{
					MaxActive:   testOptions.MaxActiveConnections,
					MaxIdle:     testOptions.MaxIdleConnections,
					IdleTimeout: testOptions.IdleTimeoutDuration,
					Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", testOptions.ConnectionAddress) },
				},
			},
		},
	}

	for idx, tt := range tests {
		s := New(tt.input)
		if s == nil {
			s = &Service{}
		}

		// note: how to check for connection address?
		assert := tt.expected.Pool.MaxActive == s.Pool.MaxActive && tt.expected.Pool.MaxIdle == s.Pool.MaxIdle &&
			tt.expected.Pool.IdleTimeout == s.Pool.IdleTimeout

		if !assert {
			t.Errorf("test #%d failed; assert: %t, input: %v, expected: %v, received: %v", idx+1, assert, tt.input, tt.expected.Pool, *s.Pool)
		}
	}
}
