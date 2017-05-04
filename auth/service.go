package auth

import (
	"errors"
)

// note @adam-hanna: can these be constants?
var (
	// ErrNoSessionKey is thrown when no key was provided for HMAC signing
	ErrNoSessionKey = errors.New("no session key")
	// ErrMalformedSession is thrown when the session value doesn't conform to expectations
	ErrMalformedSession = errors.New("malformed session")
	// ErrInvalidSession the signature included with the session can't be verified with the provided session key \
	// or is malformed in some way
	ErrInvalidSession = errors.New("invalid session")
)

// Service performs signing and verification actions using HMAC
type Service struct {
	options Options
}

// Options defines the behavior of the auth service
type Options struct {
	// Key is a slice of bytes for performing HMAC signing and verification operations
	Key []byte
}

// New returns a new auth service
func New(options Options) (*Service, error) {
	// note @adam-hanna: should we perform other checks like min/max length?
	if len(options.Key) == 0 {
		return nil, ErrNoSessionKey
	}
	return &Service{
		options: options,
	}, nil
}

// SignAndBase64Encode signs the sessionID with the key and returns a base64 encoded string
func (s *Service) SignAndBase64Encode(sessionID string) (string, error) {
	userSessionIDBytes := []byte(sessionID)
	signedBytes := signHMAC(&userSessionIDBytes, &s.options.Key)

	// append the signature to the session id
	sessionValBytes := make([]byte, len(userSessionIDBytes)+len(signedBytes))
	sessionValBytes = append(userSessionIDBytes, signedBytes...)

	return string(encode(sessionValBytes)[:]), nil
}

// VerifyAndDecode takes in a signed session string and returns a sessionID, only if the signed string passes
// auth verification.
func (s *Service) VerifyAndDecode(signed string) (string, error) {
	decodedSessionValueBytes, err := decode([]byte(signed))
	if err != nil {
		return "", err
	}

	// note: session uuid's are always 36 bytes long. This will make it difficult to switch to a new uuid algorithm!
	if len(decodedSessionValueBytes) <= 36 {
		return "", ErrInvalidSession
	}
	sessionIDBytes := decodedSessionValueBytes[:36]
	hmacBytes := decodedSessionValueBytes[36:]

	// verify the hmac signature
	verified := verifyHMAC(&sessionIDBytes, &hmacBytes, &s.options.Key)
	if !verified {
		return "", ErrInvalidSession
	}

	return string(sessionIDBytes[:]), nil
}
