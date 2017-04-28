package auth

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"errors"
)

var (
	// ErrBase64Encode is thrown when a byte slice could not be base64 encoded
	ErrBase64Encode = errors.New("Base64 encoding failed")
	// ErrBase64Decode is thrown when a byte slice could not be base64 decoded
	ErrBase64Decode = errors.New("Base64 decoding failed")
)

// Thanks! https://github.com/gorilla/securecookie
// encode encodes a value using base64.
func encode(value []byte) []byte {
	encoded := make([]byte, base64.URLEncoding.EncodedLen(len(value)))
	base64.URLEncoding.Encode(encoded, value)
	return encoded
}

// decode decodes a cookie using base64.
func decode(value []byte) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(value)))
	b, err := base64.URLEncoding.Decode(decoded, value)
	if err != nil {
		return nil, ErrBase64Decode
	}
	return decoded[:b], nil
}

func signHMAC(message, key *[]byte) []byte {
	mac := hmac.New(sha512.New, *key)
	mac.Write(*message)
	return mac.Sum(nil)
}

func verifyHMAC(message, messageMAC, key *[]byte) bool {
	mac := hmac.New(sha512.New, *key)
	mac.Write(*message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(*messageMAC, expectedMAC)
}
