package token

import "fmt"

// Common error messages
var (
	ErrInvalidToken = fmt.Errorf("[token] invalid or missing token")
)

// Error wrapping helper
func wrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[token] "+format+": %w", append(args, err)...)
}
