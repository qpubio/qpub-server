package security

import (
	"fmt"
)

// Common error messages
var (
	// Generator errors
	ErrInvalidLength  = fmt.Errorf("[security] generator: length must be positive")
	ErrEmptyCharset   = fmt.Errorf("[security] generator: charset cannot be empty")
	ErrGenerateRandom = fmt.Errorf("[security] generator: failed to generate random number")

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
