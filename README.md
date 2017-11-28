[![Build Status](https://travis-ci.org/adam-hanna/sessions.svg)](https://travis-ci.org/adam-hanna/sessions) [![Coverage Status](https://coveralls.io/repos/github/adam-hanna/sessions/badge.svg)](https://coveralls.io/github/adam-hanna/sessions) [![Go Report Card](https://goreportcard.com/badge/github.com/adam-hanna/sessions)](https://goreportcard.com/report/github.com/adam-hanna/sessions) [![GoDoc](https://godoc.org/github.com/adam-hanna/sessions?status.svg)](https://godoc.org/github.com/adam-hanna/sessions)

# Sessions
A dead simple, highly performant, highly customizable sessions service for go http servers.

By default, the service stores sessions in redis, and transports sessions to clients in cookies. However, these are easily customizeable. For instance, the storage interface only implements three methods:

~~~go
// ServiceInterface defines the behavior of the session store
type ServiceInterface interface {
	SaveUserSession(userSession *user.Session) error
	DeleteUserSession(sessionID string) error
	FetchValidUserSession(sessionID string) (*user.Session, error)
}
~~~

**README Contents:**

1. [Quickstart](https://github.com/adam-hanna/sessions#quickstart)
2. [Performance](https://github.com/adam-hanna/sessions#performance)
3. [Design](https://github.com/adam-hanna/sessions#design)
4. [API](https://github.com/adam-hanna/sessions#api)
5. [Test Coverage](https://github.com/adam-hanna/sessions#test-coverage)
6. [Example](https://github.com/adam-hanna/sessions#example)
7. [License](https://github.com/adam-hanna/sessions#license)

## Quickstart
~~~go
var sesh *sessions.Service

// issue a new session and write the session to the ResponseWriter
userSession, err := sesh.IssueUserSession("fakeUserID", "{\"foo\":\"bar\"}", w)
if err != nil {
	log.Printf("Err issuing user session: %v\n", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	return
}

...

// Fetch a pointer to a valid user session from a request. A nil pointer indicates no or invalid session
userSession, err := sesh.GetUserSession(r)
if err != nil {
	log.Printf("Err fetching user session: %v\n", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	return
}
// nil session pointers indicate a 401 unauthorized
if userSession == nil {
	http.Error(w, "Unathorized", http.StatusUnauthorized)
	return
}

...

// Extend session expiry. Note that session expiry's need to be manually extended
if err := sesh.ExtendUserSession(userSession, r, w); err != nil {
	log.Printf("Err extending user session: %v\n", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	return
}

...

// Invalidate a user session, deleting it from redis and expiring the cookie on the ResponseWriter
if err := sesh.ClearUserSession(userSession, w); err != nil {
	log.Printf("Err clearing user session: %v\n", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	return
}
~~~

## Performance
Benchmarks require a redis-server running. Set the `REDIS_URL` environment variable, otherwise the benchmarks look for ":6379".

YMMV
~~~ bash
$ (cd benchmark && go test -bench=.)

setting up benchmark tests
BenchmarkBaseServer-2              20000             72479 ns/op
BenchmarkValidSession-2            10000            151650 ns/op
PASS
shutting down benchmark tests
ok      github.com/adam-hanna/sessions/benchmark        3.727s
~~~

## Design
By default, the service stores sessions in redis, and transports hashed sessionIDs to clients in cookies. However, these are easily customizeable through the creation of custom structs that implement the interface.

The general flow of the session service is as follows:

1. Create [store](https://godoc.org/github.com/adam-hanna/sessions/store), [auth](https://godoc.org/github.com/adam-hanna/sessions/auth) and [transport](https://godoc.org/github.com/adam-hanna/sessions/transport) services by calling their respective `New(...)` functions (or create your own custom services that implement the service's interface methods). Then pass these services to the `sessions.New(...)` constructor.
2. After a user logs in, call the `sessions.IssueUserSession(...)` function. This function first creates a new `user.Session`. SessionIDs are [RFC 4122 version 4 uuids](https://github.com/pborman/uuid). Next, the service hashes the sessionID with the provided key. The hashing algorithm is SHA-512, and therefore [the key used should be between 64 and 128 bytes](https://tools.ietf.org/html/rfc2104#section-3). Then, the service stores the session in redis and finally writes the hashed sessionID to the response writer in a cookie. Sessions written to the redis db utilize `EXPIREAT` to automatically destory expired sessions.
3. To check if a valid session was included in a request, use the `sessions.GetUserSession(...)` function. This function grabs the hashed sessionID from the session cookie, verifies the HMAC signature and finally looks up the session in the redis db. If the session is expired, or fails HMAC signature verification, this function will return a nil pointer to a user session. If the session is valid, and you'd like to extend the session's expiry, you can then call `session.ExtendUserSession(...)`. Session expiry's are never automatically extended, only through calling this function will the session's expiry be extended.
4. When a user logs out, call the `sessions.ClearUserSession(...)` function. This function destroys the session in the db and also destroys the cookie on the ResponseWriter.

## API
### [user.Session](https://godoc.org/github.com/adam-hanna/sessions/user#Session)
~~~go
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	JSON      string
}
~~~
Session is the struct that is used to store session data. The JSON field allows you to set any custom information you'd like. See the [example](https://github.com/adam-hanna/sessions#example)

### [IssueUserSession](https://godoc.org/github.com/adam-hanna/sessions#IssueUserSession)
~~~ go
func (s *Service) IssueUserSession(userID string, json string, w http.ResponseWriter) (*user.Session, error)
~~~
IssueUserSession grants a new user session, writes that session info to the store and writes the session on the http.ResponseWriter.

This method should be called when a user logs in, for example.

### [ClearUserSession](https://godoc.org/github.com/adam-hanna/sessions#ClearUserSession)
~~~go
func (s *Service) ClearUserSession(userSession *user.Session, w http.ResponseWriter) error
~~~
ClearUserSession is used to remove the user session from the store and clear the cookies on the ResponseWriter.

This method should be called when a user logs out, for example.

### [GetUserSession](https://godoc.org/github.com/adam-hanna/sessions#GetUserSession)
~~~go
func (s *Service) GetUserSession(r *http.Request) (*user.Session, error)
~~~
GetUserSession returns a user session from the hashed sessionID included in the request. This method only returns valid sessions. Therefore, sessions that have expired or that fail signature verification will return a nil pointer.

### [ExtendUserSession](https://godoc.org/github.com/adam-hanna/sessions#ExtendUserSession)
~~~go
func (s *Service) ExtendUserSession(userSession *user.Session, r *http.Request, w http.ResponseWriter) error
~~~
ExtendUserSession extends the ExpiresAt of a session by the Options.ExpirationDuration

Note that this function must be called, manually! Extension of user session expiry's does not happen automatically!

## Testing Coverage
~~~bash
ok      github.com/adam-hanna/sessions			9.012s  coverage: 94.1% of statements
ok      github.com/adam-hanna/sessions/auth		0.003s  coverage: 100.0% of statements
ok      github.com/adam-hanna/sessions/store		0.006s  coverage: 85.4% of statements
ok      github.com/adam-hanna/sessions/benchmark	0.004s  coverage: 0.0% of statements [no tests to run]
ok      github.com/adam-hanna/sessions/transport	0.004s  coverage: 95.2% of statements
ok      github.com/adam-hanna/sessions/user		0.003s  coverage: 100.0% of statements
~~~

Tests are broken down into three categories: unit, integration and e2e. Integration and e2e tests require a connection to a redis server. The connection address can be set in the `REDIS_URL` environment variable. The default is ":6379".

To run all tests, simply:
~~~
$ go test -tags="unit integration e2e" ./...

// or
$ make test

// or
$ make test-cover-html && go tool cover -html=coverage-all.out
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
The following example is a demonstration of using the session service along with a CSRF code to check for authentication. The CSRF code is stored in the user.Session JSON field.

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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	myJSON := SessionJSON{
		CSRF: csrf,
	}
	JSONBytes, err := json.Marshal(myJSON)
	if err != nil {
		log.Printf("Err marhsalling json: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	userSession, err := sesh.IssueUserSession("fakeUserID", string(JSONBytes[:]), w)
	if err != nil {
		log.Printf("Err issuing user session: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
	userSession, err := sesh.GetUserSession(r)
	if err != nil {
		log.Printf("Err fetching user session: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// nil session pointers indicate a 401 unauthorized
	if userSession == nil {
		http.Error(w, "Unathorized", http.StatusUnauthorized)
		return
	}
	log.Printf("In require; user session expiration before extension: %v\n", userSession.ExpiresAt.UTC())

	myJSON := SessionJSON{}
	if err := json.Unmarshal([]byte(userSession.JSON), &myJSON); err != nil {
		log.Printf("Err unmarshalling json: %v\n", err)
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
	if err = sesh.ExtendUserSession(userSession, r, w); err != nil {
		log.Printf("Err extending user session: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// nil session pointers indicate a 401 unauthorized
	if userSession == nil {
		http.Error(w, "Unathorized", http.StatusUnauthorized)
		return
	}

	log.Printf("In clear; session: %v\n", userSession)

	myJSON := SessionJSON{}
	if err := json.Unmarshal([]byte(userSession.JSON), &myJSON); err != nil {
		log.Printf("Err unmarshalling json: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

	if err = sesh.ClearUserSession(userSession, w); err != nil {
		log.Printf("Err clearing user session: %v\n", err)
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

func main() {
	seshStore := store.New(store.Options{})

	// e.g. `$ openssl rand -base64 64`
	seshAuth, err := auth.New(auth.Options{
		Key: []byte("DOZDgBdMhGLImnk0BGYgOUI+h1n7U+OdxcZPctMbeFCsuAom2aFU4JPV4Qj11hbcb5yaM4WDuNP/3B7b+BnFhw=="),
	})
	if err != nil {
		log.Fatal(err)
	}

	seshTransport := transport.New(transport.Options{
		HTTPOnly: true,
		Secure:   false, // note: can't use secure cookies in development!
	})

	sesh = sessions.New(seshStore, seshAuth, seshTransport, sessions.Options{})

	http.HandleFunc("/issue", issueSession)
	http.HandleFunc("/require", requiresSession)
	http.HandleFunc("/clear", clearSession) // also requires a valid session

	log.Println("Listening on localhost:3000")
	log.Fatal(http.ListenAndServe("127.0.0.1:3000", nil))
}

// thanks
// https://astaxie.gitbooks.io/build-web-application-with-golang/en/06.2.html#unique-session-ids
func generateKey() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
~~~

## License
~~~
The MIT License (MIT)

Copyright (c) 2017 Adam Hanna

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
~~~