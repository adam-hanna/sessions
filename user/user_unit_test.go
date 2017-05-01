// +build unit

package user

import (
	"strings"
	"testing"
	"time"
)

// TestNew tests the New func
func TestNew(t *testing.T) {
	var tests = []struct {
		inputUserID   string
		inputJSON     string
		inputDuration time.Duration
		expected      Session
	}{
		{
			"testID",
			"testJSON",
			10 * time.Second,
			Session{
				ID:        "", // note: tested elsewhere
				UserID:    "testID",
				ExpiresAt: time.Now().Add(10 * time.Second).UTC(),
				JSON:      "testJSON",
			},
		},
	}

	for idx, tt := range tests {
		a := New(tt.inputUserID, tt.inputJSON, tt.inputDuration)
		if a == nil {
			a = &Session{}
		}

		if a.UserID != tt.inputUserID || a.JSON != tt.inputJSON || !testSessionID(a.ID) || !testExpiresAt(tt.inputDuration, a.ExpiresAt) {
			t.Errorf("test #%d failed; inputUserID: %s, inputJSON: %s, inputDuration: %v, expected session: %v, received session: %v", idx+1, tt.inputUserID, tt.inputJSON, tt.inputDuration, tt.expected, *a)
		}
	}
}

func testExpiresAt(inputDuration time.Duration, actualExpiresAt time.Time) bool {
	t1 := time.Now().Add(inputDuration).UTC()
	return actualExpiresAt.Sub(t1) < 1*time.Second
}

func testSessionID(sessionID string) bool {
	// eg 5f4cd331-c869-4871-bb41-76b726df9937
	parts := strings.Split(sessionID, "-")
	return len([]byte(sessionID)) == 36 && len([]rune(sessionID)) == 36 && strings.Count(sessionID, "-") == 4 &&
		len(parts) == 5 && len([]rune(parts[0])) == 8 && len([]rune(parts[1])) == 4 && len([]rune(parts[2])) == 4 &&
		len([]rune(parts[3])) == 4 && len([]rune(parts[4])) == 12
}
