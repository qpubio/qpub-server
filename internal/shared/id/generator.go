package id

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	shared "github.com/qpubio/qpub-server/internal/config/shared"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"strconv"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

type Generator struct {
	hashLength  int
	entropy     *ulid.MonotonicEntropy
	entropyLock sync.Mutex
}

func NewGenerator(cfg *shared.ID) (*Generator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, wrapError(err, "invalid config")
	}

	return &Generator{
		hashLength: cfg.HashLength,
		entropy:    ulid.Monotonic(rand.Reader, 0),
	}, nil
}

// ParseInt parses an integer from a string
func (g *Generator) ParseInt(s string) (Int, error) {
	return strconv.Atoi(s)
}

// MustParseInt parses an integer from a string and panics on error
func (g *Generator) MustParseInt(s string) Int {
	id, err := g.ParseInt(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse int ID %s: %v", s, err))
	}
	return id
}

// HashID generates a public identifier from an integer using base62 encoding
func (g *Generator) HashID(intID Int) Hash {
	return encodeBase62Fixed(intID, g.hashLength)
}

// ParseHashID parses a hash back to the original integer
func (g *Generator) ParseHashID(hash Hash) (Int, error) {
	if len(hash) != g.hashLength {
		return 0, wrapError(ErrInvalidHash, fmt.Sprintf("expected length %d, got %d", g.hashLength, len(hash)))
	}

	id, err := decodeBase62(hash)
	if err != nil {
		return 0, wrapError(ErrInvalidHash, err.Error())
	}
	return id, nil
}

// MustParseHashID parses a hash and panics on error
func (g *Generator) MustParseHashID(hash Hash) Int {
	id, err := g.ParseHashID(hash)
	if err != nil {
		panic(fmt.Sprintf("failed to parse hash ID %s: %v", hash, err))
	}
	return id
}

// NewULID generates a new ULID for distributed resources
func (g *Generator) NewULID() ULID {
	g.entropyLock.Lock()
	defer g.entropyLock.Unlock()
	id := ulid.MustNew(ulid.Timestamp(clock.Now()), g.entropy)
	return base64.RawURLEncoding.EncodeToString(id.Bytes())
}

// ParseULID parses a base64 encoded ULID string
func (g *Generator) ParseULID(s ULID) (ulid.ULID, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return ulid.ULID{}, wrapError(ErrInvalidULID, err.Error())
	}
	return ulid.ULID(bytes), nil
}

// ULIDTime extracts time from a ULID string
func (g *Generator) ULIDTime(s ULID) (time.Time, error) {
	id, err := g.ParseULID(s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(id.Time())/1000, 0), nil
}

// Base62 alphabet (0-9, A-Z, a-z)
const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// encodeBase62Fixed encodes an integer to base62 with exactly the specified length
// For integers that would naturally encode to fewer characters, we add a deterministic offset
func encodeBase62Fixed(num int, length int) string {
	if num < 0 {
		return ""
	}

	// Convert to base62
	result := make([]byte, 0, length)
	n := num

	if n == 0 {
		result = append(result, base62Alphabet[0])
	} else {
		for n > 0 {
			result = append([]byte{base62Alphabet[n%62]}, result...)
			n /= 62
		}
	}

	// Pad with leading zeros if needed
	for len(result) < length {
		result = append([]byte{base62Alphabet[0]}, result...)
	}

	// Truncate if too long (shouldn't happen for our use case)
	if len(result) > length {
		result = result[len(result)-length:]
	}

	return string(result)
}

// decodeBase62 decodes a base62 string back to an integer
func decodeBase62(encoded string) (int, error) {
	result := 0
	base := len(base62Alphabet)

	for _, char := range encoded {
		// Find the index of the character in the alphabet
		index := -1
		for i, c := range base62Alphabet {
			if c == char {
				index = i
				break
			}
		}

		if index == -1 {
			return 0, fmt.Errorf("invalid character in base62 string: %c", char)
		}

		result = result*base + index
	}

	return result, nil
}
