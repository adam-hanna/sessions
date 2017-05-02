package benchmark

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/adam-hanna/sessions"
	"github.com/adam-hanna/sessions/auth"
	"github.com/adam-hanna/sessions/store"
	"github.com/adam-hanna/sessions/transport"
)

var (
	issuedSessionIDs []string

	sesh      *sessions.Service
	seshStore *store.Service

	issueSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userSession, seshErr := sesh.IssueUserSession("fakeUserID", "", w)
		if seshErr != nil {
			if testing.Verbose() {
				log.Printf("Err issuing user session: %v\n", seshErr)
			}
			http.Error(w, seshErr.Err.Error(), seshErr.Code)
			return
		}

		// we need to remove these from redis during testing shutdown
		issuedSessionIDs = append(issuedSessionIDs, userSession.ID)

		w.WriteHeader(http.StatusOK)
	})

	requiresSession = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, seshErr := sesh.GetUserSession(r)
		if seshErr != nil {
			if testing.Verbose() {
				log.Printf("Err fetching user session: %v\n", seshErr)
			}
			http.Error(w, seshErr.Err.Error(), seshErr.Code)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
)

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		log.Fatal("Err setting up benchmark tests", err)
	}

	code := m.Run()

	err = shutdown()
	if err != nil {
		log.Fatal("Err shutting down benchmark tests", err)
	}

	os.Exit(code)
}

func setup() error {
	log.Println("setting up benchmark tests")

	// set up the session service
	seshStore = store.New(store.Options{})

	// e.g. `$ openssl rand -base64 64`
	authKey := "DOZDgBdMhGLImnk0BGYgOUI+h1n7U+OdxcZPctMbeFCsuAom2aFU4JPV4Qj11hbcb5yaM4WDuNP/3B7b+BnFhw=="
	authOptions := auth.Options{
		Key: []byte(authKey),
	}
	seshAuth, customErr := auth.New(authOptions)
	if customErr != nil {
		return customErr.Err
	}

	transportOptions := transport.Options{
		Secure: false, // note: can't use secure cookies in development!
	}
	seshTransport := transport.New(transportOptions)

	seshOptions := sessions.Options{
		ExpirationDuration: 3 * 24 * time.Hour,
	}
	sesh = sessions.New(seshStore, seshAuth, seshTransport, seshOptions)

	// make sure that we can connect
	c := seshStore.Pool.Get()
	defer c.Close()

	_, err := c.Do("PING")
	if err != nil {
		return err
	}

	return nil
}

func shutdown() error {
	log.Println("shutting down benchmark tests")

	c := seshStore.Pool.Get()
	defer c.Close()

	aLongTimeAgo := time.Now().Add(-1000 * time.Hour)

	for idx := range issuedSessionIDs {
		_, err := c.Do("EXPIREAT", issuedSessionIDs[idx], aLongTimeAgo.Unix())
		if err != nil {
			return errors.New("Could not delete issued session id. Error: " + err.Error())
		}
	}

	return nil
}

func BenchmarkBaseServer(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		b.Fatalf("Couldn't build request; Err: %v", err)
	}

	tr := &http.Transport{}
	defer tr.CloseIdleConnections()
	cl := &http.Client{
		Transport: tr,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := cl.Do(req)
		if err != nil {
			b.Fatal("Get:", err)
		}
		if res.StatusCode != 200 {
			b.Fatalf("Wanted 200 status code, received: %d\n", res.StatusCode)
		}
	}
}

func BenchmarkValidSession(b *testing.B) {
	// set up the test servers
	issueServer := httptest.NewServer(issueSession)
	defer issueServer.Close()

	requireServer := httptest.NewServer(requiresSession)
	defer requireServer.Close()

	// now let's get a valid session
	res, err := http.Get(issueServer.URL)
	if err != nil {
		b.Errorf("Couldn't send request to test server; Err: %v\n", err)
	}

	if res.StatusCode != 200 {
		b.Fatalf("Expected unathorized (200), received: %d\n", res.StatusCode)
	}

	// now let's send to the require server
	// first, grab the session cookie
	rc := res.Cookies()
	var sessionCookieIndex int
	for i, cookie := range rc {
		if cookie.Name == "session" {
			sessionCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", requireServer.URL, nil)
	if err != nil {
		b.Fatalf("Couldn't build request; Err: %v\n", err)
	}

	req.AddCookie(rc[sessionCookieIndex])

	tr := &http.Transport{}
	defer tr.CloseIdleConnections()
	cl := &http.Client{
		Transport: tr,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := cl.Do(req)
		if err != nil {
			b.Fatal("Get:", err)
		}
		if res.StatusCode != 200 {
			b.Fatalf("Wanted 200 status code, received: %d\n", res.StatusCode)
		}
	}
}
