// +build unit

package sessions

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
		{Options{}, Options{ExpirationDuration: DefaultExpirationDuration}},
		{Options{ExpirationDuration: 1 * time.Second}, Options{ExpirationDuration: 1 * time.Second}},
	}

	for idx, tt := range tests {
		setDefaultOptions(&tt.input)
		assert := reflect.DeepEqual(tt.expected, tt.input)

		if !assert {
			t.Errorf("test #%d failed; assert: %t, expected: %v, received: %v", idx+1, assert, tt.expected, tt.input)
		}
	}
}
