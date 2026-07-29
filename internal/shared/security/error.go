package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Common error messages
var (
	// Generator errors
	ErrInvalidLength  = fmt.Errorf("[security] generator: length must be positive")
	ErrEmptyCharset   = fmt.Errorf("[security] generator: charset cannot be empty")
	ErrGenerateRandom = fmt.Errorf("[security] generator: failed to generate random number")

	// Password errors
	ErrEmptyPassword    = fmt.Errorf("[security] password: cannot be empty")
	ErrInvalidCost      = fmt.Errorf("[security] password: bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	ErrHashPassword     = fmt.Errorf("[security] password: failed to generate hash")
	ErrPasswordMismatch = fmt.Errorf("[security] password: does not match hash")

	// Secret errors
	ErrSecretLength = fmt.Errorf("[security] secret: length must be positive")
)

// Error wrapping helpers
func wrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[security] "+format+": %w", append(args, err)...)
}
