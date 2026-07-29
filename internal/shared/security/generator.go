package security

import (
	"crypto/rand"
	"math/big"
)

// Generator provides secure random value generation
type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates a secure random string using the provided charset
func (g *Generator) Generate(length int, charset string) (string, error) {
	if length <= 0 {
		return "", ErrInvalidLength
	}
	if charset == "" {
		return "", ErrEmptyCharset
	}

	result := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", wrapError(err, "generator: failed to generate random number")
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}
