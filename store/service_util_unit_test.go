// +build unit

package store

import (
	"reflect"
	"testing"
	"time"
)

// TestSetDefaultOptions tests the setDefaultOptions function
func TestSetDefaultOptions(t *testing.T) {
	var tests = []struct {
		input    Options
		expected Options
	}{
		{Options{}, Options{MaxIdleConnections: DefaultMaxIdleConnections, ConnectionAddress: DefaultConnectionAddress, IdleTimeoutDuration: DefaultIdleTimeoutDuration}},
		{Options{ConnectionAddress: "test", MaxIdleConnections: 5, MaxActiveConnections: 5, IdleTimeoutDuration: 1 * time.Second}, Options{ConnectionAddress: "test", MaxIdleConnections: 5, MaxActiveConnections: 5, IdleTimeoutDuration: 1 * time.Second}},
	}

	for idx, tt := range tests {
		setDefaultOptions(&tt.input)
		assert := reflect.DeepEqual(tt.expected, tt.input)

		if !assert {
			t.Errorf("test #%d failed; assert: %t, expected: %v, received: %v\n", idx+1, assert, tt.expected, tt.input)
		}
	}
}
