// +build e2e

package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/adam-hanna/sessions/auth"
	"github.com/adam-hanna/sessions/store"
	"github.com/adam-hanna/sessions/transport"
)

// SessionJSON is used for marshalling and unmarshalling custom session json information.
// We're using it as an opportunity to tie csrf strings to sessions to prevent csrf attacks
type SessionJSON struct {
	CSRF string `json:"csrf"`
}

var (
	issuedSessionIDs []string

	sesh      *Service
	seshStore *store.Service

	issueSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csrf, err := generateKey()
		if err != nil {
			if testing.Verbose() {
				log.Printf("Err generating csrf: %v\n", err)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		myJSON := SessionJSON{
			CSRF: csrf,
		}
		JSONBytes, err := json.Marshal(myJSON)
		if err != nil {
			if testing.Verbose() {
				log.Printf("Err generating json: %v\n", err)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		userSession, err := sesh.IssueUserSession("fakeUserID", string(JSONBytes[:]), w)
		if err != nil {
			if testing.Verbose() {
				log.Printf("Err issuing user session: %v\n", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if testing.Verbose() {
			log.Printf("In issue; user's session: %v\n", userSession)
		}

		// we need to remove these from redis during testing shutdown
		issuedSessionIDs = append(issuedSessionIDs, userSession.ID)

		// note: we set the csrf in a cookie, but look for it in request headers
		csrfCookie := http.Cookie{
			Name:     "csrf",
			Value:    csrf,
			Expires:  userSession.ExpiresAt,
			Path:     "/",
			HttpOnly: false,
			Secure:   false, // note: can't use secure cookies in development
		}
		http.SetCookie(w, &csrfCookie)

		w.WriteHeader(http.StatusOK)
	})

	requiresSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userSession, err := sesh.GetUserSession(r)
		if err != nil {
			if testing.Verbose() {
				log.Printf("Err fetching user session: %v\n", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if userSession == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if testing.Verbose() {
			log.Printf("In require; user session expiration before extension: %v\n", userSession.ExpiresAt.UTC())
		}

		myJSON := SessionJSON{}
		if err := json.Unmarshal([]byte(userSession.JSON), &myJSON); err != nil {
			if testing.Verbose() {
				log.Printf("Err issuing unmarshalling json: %v\n", err)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if testing.Verbose() {
			log.Printf("In require; user's custom json: %v\n", myJSON)
		}

		// note: we set the csrf in a cookie, but look for it in request headers
		csrf := r.Header.Get("X-CSRF-Token")
		if csrf != myJSON.CSRF {
			if testing.Verbose() {
				log.Println("Unauthorized! CSRF token doesn't match user session")
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// note that session expiry's need to be manually extended
		if err = sesh.ExtendUserSession(userSession, r, w); err != nil {
			if testing.Verbose() {
				log.Printf("Err fetching user session: %v\n", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if testing.Verbose() {
			log.Printf("In require; users session expiration after extension: %v\n", userSession.ExpiresAt.UTC())
		}

		// need to extend the csrf cookie, too
		csrfCookie := http.Cookie{
			Name:     "csrf",
			Value:    csrf,
			Expires:  userSession.ExpiresAt,
			Path:     "/",
			HttpOnly: false,
			Secure:   false, // note: can't use secure cookies in development
		}
		http.SetCookie(w, &csrfCookie)

		w.WriteHeader(http.StatusOK)
	})

	clearSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userSession, err := sesh.GetUserSession(r)
		if err != nil {
			if testing.Verbose() {
				log.Printf("Err fetching user session: %v\n", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if userSession == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if testing.Verbose() {
			log.Printf("In clear; session: %v\n", userSession)
		}

		myJSON := SessionJSON{}
		if err := json.Unmarshal([]byte(userSession.JSON), &myJSON); err != nil {
			if testing.Verbose() {
				log.Printf("Err issuing unmarshalling json: %v\n", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if testing.Verbose() {
			log.Printf("In require; user's custom json: %v\n", myJSON)
		}

		// note: we set the csrf in a cookie, but look for it in request headers
		csrf := r.Header.Get("X-CSRF-Token")
		if csrf != myJSON.CSRF {
			if testing.Verbose() {
				log.Println("Unauthorized! CSRF token doesn't match user session")
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if err = sesh.ClearUserSession(userSession, w); err != nil {
			if testing.Verbose() {
				log.Printf("Err clearing user session: %v\n", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// need to clear the csrf cookie, too
		aLongTimeAgo := time.Now().Add(-1000 * time.Hour)
		csrfCookie := http.Cookie{
			Name:     "csrf",
			Value:    "",
			Expires:  aLongTimeAgo,
			Path:     "/",
			HttpOnly: false,
			Secure:   false, // note: can't use secure cookies in development
		}
		http.SetCookie(w, &csrfCookie)

		w.WriteHeader(http.StatusOK)
	})
)

func recoverHandler(next http.Handler) http.Handler {
	// this catches any errors and returns an internal server error to the client
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				if testing.Verbose() {
					log.Printf("Recovered! Panic: %+v\n", err)
				}
				http.Error(w, http.StatusText(500), 500)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func generateKey() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		log.Fatal("Err setting up e2e tests", err)
	}

	code := m.Run()

	if err := shutdown(); err != nil {
		log.Fatal("Err shutting down e2e tests", err)
	}

	os.Exit(code)
}

func setup() error {
	log.Println("setting up e2e tests")

	// set up the session service
	seshStore = store.New(store.Options{})

	// e.g. `$ openssl rand -base64 64`
	seshAuth, err := auth.New(auth.Options{
		Key: []byte("DOZDgBdMhGLImnk0BGYgOUI+h1n7U+OdxcZPctMbeFCsuAom2aFU4JPV4Qj11hbcb5yaM4WDuNP/3B7b+BnFhw=="),
	})
	if err != nil {
		return err
	}

	seshTransport := transport.New(transport.Options{
		Secure: false, // note: can't use secure cookies in development!
	})

	seshOptions := Options{
		ExpirationDuration: 5 * time.Second,
	}
	sesh = New(seshStore, seshAuth, seshTransport, seshOptions)

	// make sure that we can connect
	c := seshStore.Pool.Get()
	defer c.Close()

	if _, err = c.Do("PING"); err != nil {
		return err
	}

	return nil
}

func shutdown() error {
	log.Println("shutting down e2e tests")

	c := seshStore.Pool.Get()
	defer c.Close()

	aLongTimeAgo := time.Now().Add(-1000 * time.Hour)

	for idx := range issuedSessionIDs {
		if _, err := c.Do("EXPIREAT", issuedSessionIDs[idx], aLongTimeAgo.Unix()); err != nil {
			return errors.New("Could not delete issued session id. Error: " + err.Error())
		}
	}

	return nil
}

// TestE2E tests the entire system
func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	log.Println("running e2e tests")

	// set up the test servers
	issueServer := httptest.NewServer(recoverHandler(issueSession))
	defer issueServer.Close()

	requireServer := httptest.NewServer(recoverHandler(requiresSession))
	defer requireServer.Close()

	clearServer := httptest.NewServer(recoverHandler(clearSession))
	defer clearServer.Close()

	// first, let's send a request to the require server without session. This should err.
	res, err := http.Get(requireServer.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if res.StatusCode != 401 {
		t.Errorf("Expected unathorized (401), received: %d\n", res.StatusCode)
	}

	// now let's get a valid session
	res, err = http.Get(issueServer.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected unathorized (200), received: %d\n", res.StatusCode)
	}

	// now let's send to the require server, without a valid csrf. This should err.
	// first, grab the csrf
	rc := res.Cookies()
	var csrf string
	var sessionCookieIndex int
	var originalExpiresAt time.Time
	for i, cookie := range rc {
		if cookie.Name == "csrf" {
			csrf = cookie.Value
		}
		if cookie.Name == "session" {
			sessionCookieIndex = i
			originalExpiresAt = cookie.Expires
		}
	}

	req, err := http.NewRequest("GET", requireServer.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v\n", err)
	}

	req.AddCookie(rc[sessionCookieIndex])

	// send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected status code 401, received: %d\n", resp.StatusCode)
	}

	// now add the csrf to the header. This should NOT err.
	req.Header.Add("X-CSRF-Token", csrf)

	// send the request
	// but first sleep so we can tell the difference between the two expiry's
	time.Sleep(2 * time.Second) // Pause
	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, received: %d\n", resp.StatusCode)
	}

	// was the expiration extended?
	rc = resp.Cookies()
	sessionCookieIndex = 0
	for i, cookie := range rc {
		if cookie.Name == "session" {
			sessionCookieIndex = i
		}
	}

	if rc[sessionCookieIndex].Expires.Sub(originalExpiresAt) <= 0 {
		t.Errorf("Expected the session cookie to be extended; original expire at: %v, new expire at: %v\n", originalExpiresAt, rc[sessionCookieIndex].Expires)
	}

	// now let's logout
	req, err = http.NewRequest("GET", clearServer.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v\n", err)
	}

	req.AddCookie(rc[sessionCookieIndex])
	req.Header.Add("X-CSRF-Token", csrf)

	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, received: %d\n", resp.StatusCode)
	}

	// now test that the session is no longer valid
	req, err = http.NewRequest("GET", requireServer.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v\n", err)
	}

	req.AddCookie(rc[sessionCookieIndex])
	req.Header.Add("X-CSRF-Token", csrf)

	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected status code 401, received: %d\n", resp.StatusCode)
	}

	// great! now let's test that we can wait for a token to expire
	// not let's get a valid session
	res, err = http.Get(issueServer.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected unathorized (200), received: %d\n", res.StatusCode)
	}

	// now let's send to the require server
	rc = res.Cookies()
	csrf = ""
	sessionCookieIndex = 0
	for i, cookie := range rc {
		if cookie.Name == "csrf" {
			csrf = cookie.Value
		}
		if cookie.Name == "session" {
			sessionCookieIndex = i
		}
	}

	req, err = http.NewRequest("GET", requireServer.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v\n", err)
	}

	req.AddCookie(rc[sessionCookieIndex])
	req.Header.Add("X-CSRF-Token", csrf)

	// send the request after waiting
	time.Sleep(sesh.options.ExpirationDuration + 2*time.Second) // Pause

	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected status code 401, received: %d\n", resp.StatusCode)
	}
}
