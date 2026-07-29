package apikey

import "fmt"

// Common error messages
var (
	ErrInvalidAPIKey     = fmt.Errorf("[apikey] invalid or missing API key")
	ErrInvalidFormat     = fmt.Errorf("[apikey] invalid format")
	ErrInvalidComponents = fmt.Errorf("[apikey] invalid components")
	ErrDecodingFailed    = fmt.Errorf("[apikey] decoding failed")
)

// Error wrapping helper
func wrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[apikey] "+format+": %w", append(args, err)...)
}
