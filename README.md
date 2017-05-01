# Go Sessions
A dead simple, highly customizable sessions service for go http servers

## Quickstart

~~~go
package main

import (
    ...
)

var sesh *sessions.Service

var issueSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	userSession, seshErr := sesh.IssueUserSession("fakeUserID", "", w)
	if seshErr != nil {
		log.Printf("Err issuing user session: %v\n", seshErr)
		http.Error(w, seshErr.Err.Error(), seshErr.Code) // seshErr is a custom err with an http code
		return
	}
	log.Printf("In issue; user's session: %v\n", userSession)

	w.WriteHeader(http.StatusOK)
})

func main() {
	seshStore := store.New(store.Options{})

	// e.g. `$ openssl rand -base64 64`
	authKey := "DOZDgBdMhGLImnk0BGYgOUI+h1n7U+OdxcZPctMbeFCsuAom2aFU4JPV4Qj11hbcb5yaM4WDuNP/3B7b+BnFhw=="
	authOptions := auth.Options{
		Key: []byte(authKey),
	}
	seshAuth, err := auth.New(authOptions)
	if err != nil {
		log.Fatal(err)
	}

	transportOptions := transport.Options{
		Secure: false, // note: can't use secure cookies in development!
	}
	seshTransport := transport.New(transportOptions)

	seshOptions := sessions.Options{}
	sesh = sessions.New(seshStore, seshAuth, seshTransport, seshOptions)

	http.HandleFunc("/issue", issueSession)

    log.Println("Listening on localhost:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
~~~

## Testing
Tests are broken down into three categories: unit, integration and e2e. Integration and e2e tests require a connection to a redis server. The connection address can be set in the `REDIS_URL` environment variable. The default is ":6379"

To run all tests, simply:
~~~
$ go test -tags="unit integration e2e" ./...
~~~

To run only tests from one of the categories:
~~~
$ go test -tags="integration" ./...
~~~

To run only unit and integration tests:
~~~
$ go test -tags="unit integration" ./...
~~~

## Example
The following example is a demonstration of using the session service along with a CSRF code to check for authentication. The CSRF code is stored in the userSession JSON field.

~~~go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/adam-hanna/sessions"
	"github.com/adam-hanna/sessions/auth"
	"github.com/adam-hanna/sessions/store"
	"github.com/adam-hanna/sessions/transport"
)

// SessionJSON is used for marshalling and unmarshalling custom session json information.
// We're using it as an opportunity to tie csrf strings to sessions to prevent csrf attacks
type SessionJSON struct {
	CSRF string `json:"csrf"`
}

var sesh *sessions.Service

var issueSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	csrf, err := generateKey()
	if err != nil {
		log.Printf("Err generating csrf: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	myJSON := SessionJSON{
		CSRF: csrf,
	}
	JSONBytes, err := json.Marshal(myJSON)
	if err != nil {
		log.Printf("Err generating json: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userSession, seshErr := sesh.IssueUserSession("fakeUserID", string(JSONBytes[:]), w)
	if seshErr != nil {
		log.Printf("Err issuing user session: %v\n", seshErr)
		http.Error(w, seshErr.Err.Error(), seshErr.Code)
		return
	}
	log.Printf("In issue; user's session: %v\n", userSession)

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

var requiresSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	userSession, seshErr := sesh.GetUserSession(r)
	if seshErr != nil {
		log.Printf("Err fetching user session: %v\n", seshErr)
		http.Error(w, seshErr.Err.Error(), seshErr.Code)
		return
	}
	log.Printf("In require; user session expiration before extension: %v\n", userSession.ExpiresAt.UTC())

	myJSON := SessionJSON{}
	if err := json.Unmarshal([]byte(userSession.JSON), &myJSON); err != nil {
		log.Printf("Err issuing unmarshalling json: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("In require; user's custom json: %v\n", myJSON)

	// note: we set the csrf in a cookie, but look for it in request headers
	csrf := r.Header.Get("X-CSRF-Token")
	if csrf != myJSON.CSRF {
		log.Printf("Unauthorized! CSRF token doesn't match user session")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// note that session expiry's need to be manually extended
	seshErr = sesh.ExtendUserSession(userSession, r, w)
	if seshErr != nil {
		log.Printf("Err fetching user session: %v\n", seshErr)
		http.Error(w, seshErr.Err.Error(), seshErr.Code)
		return
	}
	log.Printf("In require; users session expiration after extension: %v\n", userSession.ExpiresAt.UTC())

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

var clearSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	userSession, err := sesh.GetUserSession(r)
	if err != nil {
		log.Printf("Err fetching user session: %v\n", err)
		http.Error(w, err.Err.Error(), err.Code)
		return
	}

	log.Printf("In clear; session: %v\n", userSession)

	myJSON := SessionJSON{}
	if err := json.Unmarshal([]byte(userSession.JSON), &myJSON); err != nil {
		log.Printf("Err issuing unmarshalling json: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("In require; user's custom json: %v\n", myJSON)

	// note: we set the csrf in a cookie, but look for it in request headers
	csrf := r.Header.Get("X-CSRF-Token")
	if csrf != myJSON.CSRF {
		log.Printf("Unauthorized! CSRF token doesn't match user session")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = sesh.ClearUserSession(userSession, w)
	if err != nil {
		log.Printf("Err clearing user session: %v\n", err)
		http.Error(w, err.Err.Error(), err.Code)
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

func main() {
	seshStore := store.New(store.Options{})

	// e.g. `$ openssl rand -base64 64`
	authKey := "DOZDgBdMhGLImnk0BGYgOUI+h1n7U+OdxcZPctMbeFCsuAom2aFU4JPV4Qj11hbcb5yaM4WDuNP/3B7b+BnFhw=="
	authOptions := auth.Options{
		Key: []byte(authKey),
	}
	seshAuth, err := auth.New(authOptions)
	if err != nil {
		log.Fatal(err)
	}

	transportOptions := transport.Options{
		Secure: false, // note: can't use secure cookies in development!
	}
	seshTransport := transport.New(transportOptions)

	seshOptions := sessions.Options{}
	sesh = sessions.New(seshStore, seshAuth, seshTransport, seshOptions)

	http.HandleFunc("/issue", issueSession)
	http.HandleFunc("/require", requiresSession)
	http.HandleFunc("/clear", clearSession) // also requires a valid session

	log.Println("Listening on localhost:3000")
	log.Fatal(http.ListenAndServe("127.0.0.1:3000", nil))
}

func generateKey() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
~~~