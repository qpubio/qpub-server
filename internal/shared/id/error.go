package id

import "fmt"

var (
	ErrInvalidHash     = fmt.Errorf("invalid hash format")
	ErrInvalidULID     = fmt.Errorf("invalid ULID format")
	ErrMissingHashSalt = fmt.Errorf("HASHID_SALT environment variable is required")
)

func wrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[id] "+format+": %w", append(args, err)...)
}
