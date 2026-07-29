package apikey

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Common error messages
var (
	ErrSecurityInit = fmt.Errorf("[apikey] security: initialization failed")
	ErrNotFound     = gorm.ErrRecordNotFound
)

// Error wrapping helper
func wrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[apikey] "+format+": %w", append(args, err)...)
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
