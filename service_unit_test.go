// +build unit

package sessions

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/adam-hanna/sessions/sessionerrs"
	"github.com/adam-hanna/sessions/user"
)

var (
	MockedTestErr = sessionerrs.Custom{Code: -1, Err: errors.New("test err")}

	mockedStore     = MockedStoreType{}
	mockedAuth      = MockedAuthType{}
	mockedTransport = MockedTransportType{}

	erredStore     = ErredStoreType{}
	erredAuth      = ErredAuthType{}
	erredTransport = ErredTransportType{}

	opts = Options{ExpirationDuration: DefaultExpirationDuration}

	inputUserID = "testID"
	inputJSON   = "testJSON"
	userSession = &user.Session{
		ID:        "fakeID",
		UserID:    inputUserID,
		JSON:      inputJSON,
		ExpiresAt: time.Now().Add(opts.ExpirationDuration).UTC(),
	}
)

type MockedAuthType struct {
}

func (a *MockedAuthType) SignAndBase64Encode(sessionID string) (string, *sessionerrs.Custom) {
	return "test", nil
}

func (a *MockedAuthType) VerifyAndDecode(signed string) (string, *sessionerrs.Custom) {
	return "test", nil
}

type ErredAuthType struct {
}

func (b *ErredAuthType) SignAndBase64Encode(sessionID string) (string, *sessionerrs.Custom) {
	return "", &MockedTestErr
}

func (b *ErredAuthType) VerifyAndDecode(signed string) (string, *sessionerrs.Custom) {
	return "", &MockedTestErr
}

type MockedStoreType struct {
}

func (c *MockedStoreType) SaveUserSession(userSession *user.Session) *sessionerrs.Custom {
	return nil
}

func (c *MockedStoreType) DeleteUserSession(sessionID string) *sessionerrs.Custom {
	return nil
}

func (c *MockedStoreType) FetchValidUserSession(sessionID string) (*user.Session, *sessionerrs.Custom) {
	return userSession, nil
}

type ErredStoreType struct {
}

func (d *ErredStoreType) SaveUserSession(userSession *user.Session) *sessionerrs.Custom {
	return &MockedTestErr
}

func (d *ErredStoreType) DeleteUserSession(sessionID string) *sessionerrs.Custom {
	return &MockedTestErr
}

func (d *ErredStoreType) FetchValidUserSession(sessionID string) (*user.Session, *sessionerrs.Custom) {
	return nil, &MockedTestErr
}

type MockedTransportType struct {
}

func (e *MockedTransportType) SetSessionOnResponse(session string, userSession *user.Session, w http.ResponseWriter) *sessionerrs.Custom {
	return nil
}

func (e *MockedTransportType) DeleteSessionFromResponse(w http.ResponseWriter) *sessionerrs.Custom {
	return nil
}

func (e *MockedTransportType) FetchSessionIDFromRequest(r *http.Request) (string, *sessionerrs.Custom) {
	return "test", nil
}

type ErredTransportType struct {
}

func (f *ErredTransportType) SetSessionOnResponse(session string, userSession *user.Session, w http.ResponseWriter) *sessionerrs.Custom {
	return &MockedTestErr
}

func (f *ErredTransportType) DeleteSessionFromResponse(w http.ResponseWriter) *sessionerrs.Custom {
	return &MockedTestErr
}

func (f *ErredTransportType) FetchSessionIDFromRequest(r *http.Request) (string, *sessionerrs.Custom) {
	return "test", &MockedTestErr
}

// TestNew tests the New function
func TestNew(t *testing.T) {
	var expectedService = Service{
		store:     &mockedStore,
		auth:      &mockedAuth,
		transport: &mockedTransport,
		options:   opts,
	}

	actualService := New(&mockedStore, &mockedAuth, &mockedTransport, Options{})
	if actualService == nil {
		actualService = &Service{}
	}

	assert := reflect.DeepEqual(expectedService, *actualService)

	if !assert {
		t.Errorf("test failed; assert: %t, expected: %v, received: %v", assert, expectedService, actualService)
	}
}

// TestIssueUserSession tests the IssueUserSession function
func TestIssueUserSession(t *testing.T) {
	var w http.ResponseWriter

	var tests = []struct {
		input               Service
		expectedUserSession *user.Session
		expectedErr         *sessionerrs.Custom
	}{
		{
			Service{
				store:     &erredStore,
				auth:      &erredAuth,
				transport: &erredTransport,
				options:   opts,
			},
			nil,
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &erredAuth,
				transport: &erredTransport,
				options:   opts,
			},
			nil,
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &mockedAuth,
				transport: &erredTransport,
				options:   opts,
			},
			userSession, // note: when transport is erred, the session, as well as an error, get returned
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &mockedAuth,
				transport: &mockedTransport,
				options:   opts,
			},
			userSession,
			nil,
		},
	}

	for idx, tt := range tests {
		var assertSession bool
		var assertErr bool
		a, e := tt.input.IssueUserSession(inputUserID, inputJSON, w)
		if a == nil {
			assertSession = a == tt.expectedUserSession
			a = &user.Session{}
		} else {
			t1 := time.Now().Add(tt.input.options.ExpirationDuration).UTC()
			assertSession = tt.expectedUserSession.UserID == a.UserID && tt.expectedUserSession.JSON == a.JSON &&
				tt.expectedUserSession.ExpiresAt.Sub(t1) < 1*time.Second
		}
		if e == nil {
			assertErr = e == tt.expectedErr
			e = &sessionerrs.Custom{}
		} else {
			assertErr = reflect.DeepEqual(*tt.expectedErr, *e)
		}

		if !assertSession || !assertErr {
			t.Errorf("test #%d failed; input service: %v, assertSession: %t, assertErr: %t, expectedSession: %v, expectedErr: %v, received session: %v, received err: %v", idx+1, tt.input, assertSession, assertErr, tt.expectedUserSession, tt.expectedErr, *a, *e)
		}
	}
}

// TestClearUserSession tests the ClearUserSession function
func TestClearUserSession(t *testing.T) {
	var w http.ResponseWriter

	var tests = []struct {
		input       Service
		expectedErr *sessionerrs.Custom
	}{
		{
			Service{
				store:     &erredStore,
				auth:      &mockedAuth,
				transport: &erredTransport,
				options:   opts,
			},
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &mockedAuth,
				transport: &erredTransport,
				options:   opts,
			},
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &mockedAuth,
				transport: &mockedTransport,
				options:   opts,
			},
			nil,
		},
	}

	for idx, tt := range tests {
		e := tt.input.ClearUserSession(userSession, w)
		assertErr := e == tt.expectedErr

		if !assertErr {
			t.Errorf("test #%d failed; input service: %v, assertErr: %t, expectedErr: %v, received err: %v", idx+1, tt.input, assertErr, tt.expectedErr, e)
		}
	}
}

// TestGetUserSession tests the GetUserSession function
func TestGetUserSession(t *testing.T) {
	r := &http.Request{}

	var tests = []struct {
		input           Service
		expectedSession *user.Session
		expectedErr     *sessionerrs.Custom
	}{
		{
			Service{
				store:     &erredStore,
				auth:      &erredAuth,
				transport: &erredTransport,
				options:   opts,
			},
			nil,
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &erredAuth,
				transport: &erredTransport,
				options:   opts,
			},
			nil,
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &mockedAuth,
				transport: &erredTransport,
				options:   opts,
			},
			nil,
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &mockedAuth,
				transport: &mockedTransport,
				options:   opts,
			},
			userSession,
			nil,
		},
	}

	for idx, tt := range tests {
		a, e := tt.input.GetUserSession(r)
		assertErr := e == tt.expectedErr
		assertSession := a == tt.expectedSession

		if !assertSession || !assertErr {
			t.Errorf("test #%d failed; input service: %v, assertSession: %t, assertErr: %t, expected session: %v, expectedErr: %v, received session: %v, received err: %v", idx+1, tt.input, assertSession, assertErr, tt.expectedSession, tt.expectedErr, a, e)
		}
	}
}

// TestGetUserSession tests the GetUserSession function
func TestExtendUserSession(t *testing.T) {
	r := &http.Request{}
	var w http.ResponseWriter

	var tests = []struct {
		input       Service
		expectedErr *sessionerrs.Custom
	}{
		{
			Service{
				store:     &erredStore,
				auth:      &mockedAuth,
				transport: &erredTransport,
				options:   opts,
			},
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &mockedAuth,
				transport: &erredTransport,
				options:   opts,
			},
			&MockedTestErr,
		},
		{
			Service{
				store:     &mockedStore,
				auth:      &mockedAuth,
				transport: &mockedTransport,
				options:   opts,
			},
			nil,
		},
	}

	for idx, tt := range tests {
		// let's use a test user session bc we don't want to mess with the one defined above
		testUserSession := &user.Session{
			ExpiresAt: time.Now().UTC(),
		}
		e := tt.input.ExtendUserSession(testUserSession, r, w)
		assertErr := e == tt.expectedErr

		newExpiresAt := time.Now().Add(tt.input.options.ExpirationDuration).UTC()
		assertExtension := newExpiresAt.Sub(testUserSession.ExpiresAt) < 1*time.Second

		if !assertExtension || !assertErr {
			t.Errorf("test #%d failed; input service: %v, assertSession: %t, assertErr: %t, expected expires at: %v, expectedErr: %v, received expires at: %v, received err: %v", idx+1, tt.input, assertExtension, assertErr, newExpiresAt, tt.expectedErr, testUserSession.ExpiresAt, e)
		}
	}
}
