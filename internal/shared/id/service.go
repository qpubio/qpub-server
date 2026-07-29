package id

import (
	"fmt"
	shared "github.com/qpubio/qpub-server/internal/config/shared"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service provides ID generation and parsing functionality
type Service interface {
	// Int methods
	ParseInt(s string) (Int, error)
	MustParseInt(s string) Int

	// Hash ID methods
	HashID(intID Int) Hash
	ParseHashID(hash Hash) (Int, error)
	MustParseHashID(hash Hash) Int

	// ULID methods
	NewULID() ULID
	ParseULID(s ULID) (ulid.ULID, error)
	ULIDTime(s ULID) (time.Time, error)
}

var (
	service Service
	once    sync.Once
)

// Init initializes the ID service singleton
func Init(cfg *shared.ID) error {
	var initErr error
	once.Do(func() {
		generator, err := NewGenerator(cfg)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize id service: %w", err)
			return
		}
		service = generator
	})
	return initErr
}

// Get returns the ID service singleton instance
func Get() Service {
	if service == nil {
		panic("id service not initialized")
	}
	return service
}

// Convenience functions using the singleton service
func ParseInt(s string) (Int, error) {
	return Get().ParseInt(s)
}

func MustParseInt(s string) Int {
	return Get().MustParseInt(s)
}

func HashID(intID Int) Hash {
	return Get().HashID(intID)
}

func ParseHashID(hash Hash) (Int, error) {
	return Get().ParseHashID(hash)
}

func MustParseHashID(hash Hash) Int {
	return Get().MustParseHashID(hash)
}

func NewULID() ULID {
	return Get().NewULID()
}

func ParseULID(s ULID) (ulid.ULID, error) {
	return Get().ParseULID(s)
}

func ULIDTime(s ULID) (time.Time, error) {
	return Get().ULIDTime(s)
}
