package auth

import (
	"errors"

	"github.com/adam-hanna/sessions/sessionerrs"
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
func New(options Options) (*Service, *sessionerrs.Custom) {
	// note @adam-hanna: should we perform other checks like min/max length?
	if len(options.Key) == 0 {
		return &Service{}, &sessionerrs.Custom{
			Code: 500,
			Err:  errors.New("no session key"),
		}
	}
	return &Service{
		options: options,
	}, nil
}

// SignAndBase64Encode signs the sessionID with they key and returns a base64 encoded string
func (s *Service) SignAndBase64Encode(sessionID string) (string, *sessionerrs.Custom) {
	userSessionIDBytes := []byte(sessionID)
	signedBytes := signHMAC(&userSessionIDBytes, &s.options.Key)

	// append the signature to the session id
	sessionValBytes := make([]byte, len(userSessionIDBytes)+len(signedBytes))
	sessionValBytes = append(userSessionIDBytes, signedBytes...)

	return string(encode(sessionValBytes)[:]), nil
}

// VerifyAndDecode takes in a signed session string and returns a sessionID, only if the signed string passes
// auth verification.
func (s *Service) VerifyAndDecode(signed string) (string, *sessionerrs.Custom) {
	sessionValueBytes := []byte(signed)
	decodedSessionValueBytes, err := decode(sessionValueBytes)
	if err != nil {
		return "", &sessionerrs.Custom{
			Code: 500,
			Err:  err,
		}
	}

	// note: session uuid's are always 36 bytes long
	if len(decodedSessionValueBytes) <= 36 {
		// note @adam-hanna: is 401 the proper http status code, here?
		return "", &sessionerrs.Custom{
			Code: 401,
			Err:  errors.New("invalid session"),
		}
	}
	sessionIDBytes := decodedSessionValueBytes[:36]
	hmacBytes := decodedSessionValueBytes[36:]
	// fmt.Printf("In auth.VerifyAndDecode\nsessionID: %s\nsig: %x\nkey: %s\n", string(sessionIDBytes[:]), string(hmacBytes[:]), string(s.options.Key[:]))

	// verify the hmac signature
	verified := verifyHMAC(&sessionIDBytes, &hmacBytes, &s.options.Key)
	if !verified {
		return "", &sessionerrs.Custom{
			Code: 401,
			Err:  errors.New("invalid session signature"),
		}
	}

	return string(sessionIDBytes[:]), nil
}
