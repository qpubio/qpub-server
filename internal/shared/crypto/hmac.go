package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign computes HMAC-SHA256 of data using the given secret key and returns the hex-encoded result.
func Sign(data []byte, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify verifies a hex-encoded HMAC-SHA256 signature against data and secret key.
// Uses constant-time comparison to prevent timing attacks.
func Verify(data []byte, secret []byte, signature string) bool {
	expected := Sign(data, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
