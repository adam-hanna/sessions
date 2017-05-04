package auth

// ServiceInterface defines the methods that are performend by the auth service
type ServiceInterface interface {
	SignAndBase64Encode(sessionID string) (string, error)
	VerifyAndDecode(signed string) (string, error)
}
