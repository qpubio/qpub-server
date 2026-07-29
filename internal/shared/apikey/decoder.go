package apikey

import (
	"encoding/base64"
	"net/url"
)

// Decoder handles API key decoding operations
type Decoder struct{}

// NewDecoder creates a new Decoder instance
func NewDecoder() *Decoder {
	return &Decoder{}
}

// DecodeKey handles both base64 and URL-encoded API keys
func (d *Decoder) DecodeKey(apiKey string, isURLEncoded bool) (string, error) {
	if isURLEncoded {
		decoded, err := url.QueryUnescape(apiKey)
		if err != nil {
			return "", wrapError(err, "URL decoding failed")
		}
		return decoded, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(apiKey)
	if err != nil {
		return "", wrapError(err, "base64 decoding failed")
	}
	return string(decoded), nil
}
