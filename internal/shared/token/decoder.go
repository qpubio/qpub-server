package token

import "net/url"

// Decoder handles token decoding operations
type Decoder struct{}

// NewDecoder creates a new Decoder instance
func NewDecoder() *Decoder {
	return &Decoder{}
}

// DecodeToken handles URL-encoded tokens
func (d *Decoder) DecodeToken(token string) (string, error) {
	decoded, err := url.QueryUnescape(token)
	if err != nil {
		return "", wrapError(err, "URL decoding failed")
	}
	return decoded, nil
}
