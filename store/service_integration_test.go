// +build integration

package store

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/adam-hanna/sessions/user"
	"github.com/garyburd/redigo/redis"
)

var (
	service          *Service
	validUserSession = &user.Session{
		ID:        "validSessionID",
		UserID:    "validUserID",
		JSON:      "validJSON",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	validUserSessionForSaving = &user.Session{
		ID:        "validSessionForSavingID",
		UserID:    "validSessionForSavingID",
		JSON:      "validSessionForSavingJSON",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	// session is invalid because it doesn't have json
	inValidUserSession = &user.Session{
		ID:     "invalidSessionID",
		UserID: "invalidUserID",
		// JSON:      "invalidJSON",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	expiredUserSession = &user.Session{
		ID:        "expiredSessionID",
		UserID:    "expiredUserID",
		JSON:      "expiredJSON",
		ExpiresAt: time.Now().Add(-100 * time.Hour),
	}
)

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		log.Fatal("Err setting up integration tests")
	}

	code := m.Run()

	if err := shutdown(); err != nil {
		log.Fatal("Err shutting down integration tests")
	}

	os.Exit(code)
}

func setup() error {
	fmt.Println("setting up redis integration tests")

	options := Options{
		ConnectionAddress: os.Getenv("REDIS_URL"),
	}
	service = New(options)
	c := service.Pool.Get()
	defer c.Close()

	// VALID USER
	if _, err := c.Do("HMSET", validUserSession.ID, "UserID", validUserSession.UserID, "JSON", validUserSession.JSON, "ExpiresAtSeconds", validUserSession.ExpiresAt.Unix()); err != nil {
		return errors.New("Could not set valid user")
	}
	if _, err := c.Do("EXPIREAT", validUserSession.ID, validUserSession.ExpiresAt.Unix()); err != nil {
		return errors.New("Could not set expiry for valid user")
	}

	// INVALID USER
	// note: the invalid user doesn't have JSON!
	if _, err := c.Do("HMSET", inValidUserSession.ID, "UserID", inValidUserSession.UserID, "ExpiresAtSeconds", inValidUserSession.ExpiresAt.Unix()); err != nil {
		return errors.New("Could not set valid user")
	}
	if _, err := c.Do("EXPIREAT", inValidUserSession.ID, inValidUserSession.ExpiresAt.Unix()); err != nil {
		return errors.New("Could not set expiry for valid user")
	}

	// EXPIRED USER
	if _, err := c.Do("HMSET", expiredUserSession.ID, "UserID", expiredUserSession.UserID, "JSON", expiredUserSession.JSON, "ExpiresAtSeconds", expiredUserSession.ExpiresAt.Unix()); err != nil {
		return errors.New("Could not set valid user")
	}
	if _, err := c.Do("EXPIREAT", expiredUserSession.ID, expiredUserSession.ExpiresAt.Unix()); err != nil {
		return errors.New("Could not set expiry for valid user")
	}

	return nil
}

func shutdown() error {
	fmt.Println("shutting down redis integration tests")

	c := service.Pool.Get()
	defer c.Close()

	aLongTimeAgo := time.Now().Add(-1000 * time.Hour)

	// VALID USER
	if _, err := c.Do("EXPIREAT", validUserSession.ID, aLongTimeAgo.Unix()); err != nil {
		return errors.New("Could not set EXPIREAT for validUserSession")
	}

	// VALID USER
	if _, err := c.Do("EXPIREAT", validUserSessionForSaving.ID, aLongTimeAgo.Unix()); err != nil {
		return errors.New("Could not set EXPIREAT for validUserSessionForSaving")
	}

	// INVALID USER
	if _, err := c.Do("EXPIREAT", inValidUserSession.ID, aLongTimeAgo.Unix()); err != nil {
		return errors.New("Could not set EXPIREAT for invaludUserSession")
	}

	// EXPIRED USER
	if _, err := c.Do("EXPIREAT", expiredUserSession.ID, aLongTimeAgo.Unix()); err != nil {
		return errors.New("Could not set EXPIREAT for expiredUserSession")
	}

	return nil
}

// TestSaveUserSession tests the SaveUserSession function
func TestSaveUserSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestSaveUserSession, an integration test")
	}

	tests := []struct {
		input         *user.Session
		expectedErr   error
		expectToExist bool
	}{
		{validUserSessionForSaving, nil, true},
		{expiredUserSession, nil, false},
	}

	c := service.Pool.Get()
	defer c.Close()

	for idx, tt := range tests {
		var assertErr bool
		var assertExist bool

		e := service.SaveUserSession(tt.input)
		assertErr = e == tt.expectedErr

		exists, err := redis.Bool(c.Do("EXISTS", tt.input.ID))
		if err != nil {
			t.Errorf("Err in test #%d; cannot check if sessionID: %s exists\n", idx, tt.input.ID)
		}

		assertExist = exists == tt.expectToExist

		if !assertErr || !assertExist {
			t.Errorf("test #%d failed; assert err: %t, assert exists: %t, received err: %v, received exists: %t, expected err: %v, expected exists: %t, input: %v\n", idx+1, assertErr, assertExist, e, exists, tt.expectedErr, tt.expectToExist, tt.input)
		}
	}
}

// TestFetchValidUserSession tests the FetchValidUserSession function
func TestFetchValidUserSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestFetchValidUserSession, an integration test")
	}

	tests := []struct {
		input               string
		expectedUserSession *user.Session
		expectedErr         error
	}{
		{validUserSession.ID, validUserSession, nil},
		{validUserSessionForSaving.ID, validUserSessionForSaving, nil},
		{expiredUserSession.ID, nil, nil},
		{inValidUserSession.ID, nil, ErrRetrievingSession},
	}

	for idx, tt := range tests {
		var assertErr bool
		var assertUserSession bool

		a, e := service.FetchValidUserSession(tt.input)
		assertErr = e == tt.expectedErr
		if a == nil {
			assertUserSession = a == tt.expectedUserSession
		} else {
			if tt.expectedUserSession != nil {
				// note: we can't use deep equal here bc the expiry time might be off by a second or so
				assertUserSession = a.ID == tt.expectedUserSession.ID && a.UserID == tt.expectedUserSession.UserID &&
					a.JSON == tt.expectedUserSession.JSON && a.ExpiresAt.Sub(tt.expectedUserSession.ExpiresAt) < 1
			}
		}

		if !assertErr || !assertUserSession {
			t.Errorf("test #%d failed; assert err: %t, assert user session: %t, received err: %v, received user session: %v, expected err: %v, expected user session: %v, input: %v\n", idx+1, assertErr, assertUserSession, e, a, tt.expectedErr, tt.expectedUserSession, tt.input)
		}
	}
}

// TestDeleteUserSession tests the DeleteUserSession function
func TestDeleteUserSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestDeleteUserSession, an integration test")
	}

	tests := []struct {
		input         string
		expectToExist bool
	}{
		{validUserSession.ID, false},
		{validUserSessionForSaving.ID, false},
		{expiredUserSession.ID, false},
		{inValidUserSession.ID, false},
	}

	c := service.Pool.Get()
	defer c.Close()

	for idx, tt := range tests {
		var assertExist bool

		e := service.DeleteUserSession(tt.input)
		if e != nil {
			t.Errorf("Err in test #%d when deleting user session, expected err to be nil, received err: %v, input: %s\n", idx, e, tt.input)
		}

		exists, err := redis.Bool(c.Do("EXISTS", tt.input))
		if err != nil {
			t.Errorf("Err in test #%d when checking exists, expected err to be nil, received err: %v, input: %s\n", idx, err, tt.input)
		}

		assertExist = exists == tt.expectToExist
		if err != nil {
			t.Errorf("Err in test #%d; assertExist: %t, expected exists: %t, received exists: %t, input: %s\n", idx, assertExist, tt.expectToExist, exists, tt.input)
		}
	}
}
