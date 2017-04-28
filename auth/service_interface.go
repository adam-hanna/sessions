package auth

import "github.com/adam-hanna/sessions/sessionerrs"

// ServiceInterface defines the methods that are performend by the auth service
type ServiceInterface interface {
	SignAndBase64Encode(sessionID string) (string, *sessionerrs.Custom)
	VerifyAndDecode(signed string) (string, *sessionerrs.Custom)
}
