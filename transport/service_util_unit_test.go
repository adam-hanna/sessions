// +build unit

package transport

import (
	"reflect"
	"testing"
)

// TestSetDefaultOptions tests the setDefaultOptions function
func TestSetDefaultOptions(t *testing.T) {
	var tests = []struct {
		input    Options
		expected Options
	}{
		{Options{}, Options{CookieName: DefaultCookieName, CookiePath: DefaultCookiePath}},
		{Options{CookieName: "test", CookiePath: "/", HTTPOnly: true, Secure: true}, Options{CookieName: "test", CookiePath: "/", HTTPOnly: true, Secure: true}},
	}

	for idx, tt := range tests {
		setDefaultOptions(&tt.input)
		assert := reflect.DeepEqual(tt.expected, tt.input)

		if !assert {
			t.Errorf("test #%d failed; assert: %t, expected: %v, received: %v", idx+1, assert, tt.expected, tt.input)
		}
	}
}
