// +build unit

package transport

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/adam-hanna/sessions/user"
)

var (
	testOptions = Options{CookieName: "test", CookiePath: "/", HTTPOnly: true, Secure: true}
	testService = Service{options: testOptions}
)

// Thanks!
// https://gist.github.com/karlseguin/5128461
type FakeResponse struct {
	headers http.Header
	body    []byte
	status  int
}

func (f FakeResponse) Write(body []byte) (int, error) {
	f.body = body
	return len(body), nil
}

func (f FakeResponse) WriteHeader(status int) {
	f.status = status
}

func (f FakeResponse) Header() http.Header {
	return f.headers
}

// TestNew tests the New function
func TestNew(t *testing.T) {
	var tests = []struct {
		input    Options
		expected Service
	}{
		{Options{}, Service{options: Options{CookieName: DefaultCookieName, CookiePath: DefaultCookiePath}}},
		{testOptions, Service{options: testOptions}},
	}

	for idx, tt := range tests {
		s := New(tt.input)
		if s == nil {
			s = &Service{}
		}
		assert := reflect.DeepEqual(tt.expected, *s)

		if !assert {
			t.Errorf("test #%d failed; assert: %t, input: %v, expected: %v, received: %v", idx+1, assert, tt.input, tt.expected, *s)
		}
	}
}

// TestSetSessionOnResponse tests the SetSessionOnResponse function
func TestSetSessionOnResponse(t *testing.T) {
	u := user.New("testID", "", 1*time.Second)
	m := make(map[string][]string, 1)
	b := make([]byte, 1)
	f := FakeResponse{m, b, 0}
	var tests = []struct {
		signedSessionID string
		userSession     *user.Session
		w               FakeResponse
	}{
		{"testSignedSessionID", u, f},
	}

	for idx, tt := range tests {
		expectedW := FakeResponse{m, b, 0}
		sessionCookie := http.Cookie{
			Name:     testService.options.CookieName,
			Value:    tt.signedSessionID,
			Expires:  tt.userSession.ExpiresAt,
			Path:     testService.options.CookiePath,
			HttpOnly: testService.options.HTTPOnly,
			Secure:   testService.options.Secure,
		}
		http.SetCookie(expectedW, &sessionCookie)

		_ = testService.SetSessionOnResponse(tt.signedSessionID, tt.userSession, tt.w)

		assert := reflect.DeepEqual(tt.w, expectedW)
		if !assert {
			t.Errorf("test #%d failed; assert: %t, w: %v, expected: %v", idx+1, assert, tt.w, expectedW)
		}
	}
}

// TestDeleteSessionFromResponse tests the DeleteSessionFromResponse function
func TestDeleteSessionFromResponse(t *testing.T) {
	m := make(map[string][]string, 1)
	b := make([]byte, 1)
	f := FakeResponse{m, b, 0}
	var tests = []struct {
		w FakeResponse
	}{
		{f},
	}

	for idx, tt := range tests {
		expectedW := FakeResponse{m, b, 0}
		aLongTimeAgo := time.Now().Add(-1000 * time.Hour)
		nullSessionCookie := http.Cookie{
			Name:     testService.options.CookieName,
			Value:    "",
			Expires:  aLongTimeAgo,
			Path:     testService.options.CookiePath,
			HttpOnly: testService.options.HTTPOnly,
			Secure:   testService.options.Secure,
		}
		http.SetCookie(expectedW, &nullSessionCookie)

		_ = testService.DeleteSessionFromResponse(tt.w)

		assert := reflect.DeepEqual(tt.w, expectedW)
		if !assert {
			t.Errorf("test #%d failed; assert: %t, w: %v, expected: %v", idx+1, assert, tt.w, expectedW)
		}
	}
}

// TestFetchSessionIDFromRequest tests the FetchSessionIDFromRequest function
func TestFetchSessionIDFromRequest(t *testing.T) {
	var tests = []struct {
		input          http.Cookie
		expectedString string
		expectedErr    error
	}{
		{http.Cookie{Name: testService.options.CookieName, Value: "testValue"}, "testValue", nil},
		{http.Cookie{Name: "badName"}, "", ErrNoSessionOnRequest},
	}

	for idx, tt := range tests {
		m := make(map[string][]string, 1)
		r := http.Request{Header: m}
		r.AddCookie(&tt.input)

		s, e := testService.FetchSessionIDFromRequest(&r)
		assert := tt.expectedErr == e

		if !assert || tt.expectedString != s {
			t.Errorf("test #%d failed; assert: %t, expectedErr: %v, err: %v, expectedString: %s, string: %s", idx+1, assert, tt.expectedErr, e, tt.expectedString, s)
		}
	}
}
