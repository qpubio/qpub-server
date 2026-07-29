package validation

import (
	"encoding/json"
	"fmt"
	"net/url"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/slug"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Rules provides common validation rules
type Rules struct {
	errors map[string]string
}

// NewRules creates a new Rules instance
func NewRules() *Rules {
	return &Rules{
		errors: make(map[string]string),
	}
}

// Required checks if a value is not empty across various types
func (r *Rules) Required(value interface{}, field string) {
	if value == nil {
		r.AddError(field, "is required")
		return
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			r.AddError(field, "is required")
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		// Numeric types are always considered non-empty
		return
	case bool:
		// Boolean types are always considered non-empty
		return
	case []string:
		if len(v) == 0 {
			r.AddError(field, "is required")
		}
	case []interface{}:
		if len(v) == 0 {
			r.AddError(field, "is required")
		}
	case map[string]interface{}:
		if len(v) == 0 {
			r.AddError(field, "is required")
		}
	case map[string]string:
		if len(v) == 0 {
			r.AddError(field, "is required")
		}
	case time.Time:
		if v.IsZero() {
			r.AddError(field, "is required")
		}
	case *time.Time:
		if v == nil || v.IsZero() {
			r.AddError(field, "is required")
		}
	default:
		// For other types, use reflection to check if it's a zero value
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Ptr && rv.IsNil() {
			r.AddError(field, "is required")
			return
		}

		// For slices, maps, and arrays
		switch rv.Kind() {
		case reflect.Slice, reflect.Map, reflect.Array:
			if rv.Len() == 0 {
				r.AddError(field, "is required")
			}
		case reflect.Struct:
			// For structs, we consider them non-empty by default
			// If you need to check if a struct is "empty", you might need custom logic
			return
		default:
			// For other types, check if it's the zero value
			if reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
				r.AddError(field, "is required")
			}
		}
	}
}

// IsPositive checks if a value is positive
func (r *Rules) IsPositive(value int, field string) {
	if value <= 0 {
		r.AddError(field, "must be positive")
	}
}

// MinLength checks if a string meets the minimum length requirement
func (r *Rules) MinLength(value string, min int, field string) {
	if len(value) < min {
		r.AddError(field, fmt.Sprintf("must be at least %d characters", min))
	}
}

// MaxLength checks if a string meets the maximum length requirement
func (r *Rules) MaxLength(value string, max int, field string) {
	if len(value) > max {
		r.AddError(field, fmt.Sprintf("must not exceed %d characters", max))
	}
}

// Email validates email format
func (r *Rules) Email(email string, field string) {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		r.AddError(field, "must be a valid email address")
	}
}

// Phone validates phone format
func (r *Rules) Phone(phone string, field string) {
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(phone) {
		r.AddError(field, "must be a valid phone number")
	}
}

// URL validates URL format
func (r *Rules) URL(value string, field string) {
	if value == "" {
		return // Skip empty URLs unless Required is also used
	}
	_, err := url.ParseRequestURI(value)
	if err != nil {
		r.AddError(field, "must be a valid URL")
	}
}

// JSON validates JSON format
func (r *Rules) JSON(value json.RawMessage, field string) {
	if !json.Valid(value) {
		r.AddError(field, "must be valid JSON")
	}
}

// Array validates if a value is an array
func (r *Rules) Array(value []string, field string) {
	if len(value) == 0 {
		r.AddError(field, "must be an array")
	}
}

// NumericValidation holds options for numeric validation
type NumericValidation struct {
	Min           *float64 // Minimum value (optional)
	Max           *float64 // Maximum value (optional)
	AllowNegative bool     // Whether to allow negative numbers
	AllowZero     bool     // Whether to allow zero
	AllowDecimals bool     // Whether to allow decimal numbers
}

// DefaultNumericValidation returns default numeric validation options
func DefaultNumericValidation() NumericValidation {
	return NumericValidation{
		AllowNegative: true,
		AllowZero:     true,
		AllowDecimals: true,
	}
}

// Number validates if a string value is a valid number with options
func (r *Rules) Number(value string, field string, opts ...NumericValidation) {
	if value == "" {
		return // Skip empty values unless Required is also used
	}

	// Use default options if none provided
	validation := DefaultNumericValidation()
	if len(opts) > 0 {
		validation = opts[0]
	}

	// Try to parse the number
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		r.AddError(field, "must be a valid number")
		return
	}

	// Check if decimals are allowed
	if !validation.AllowDecimals && strings.Contains(value, ".") {
		r.AddError(field, "must be a whole number")
		return
	}

	// Check if negative numbers are allowed
	if !validation.AllowNegative && num < 0 {
		r.AddError(field, "cannot be negative")
		return
	}

	// Check if zero is allowed
	if !validation.AllowZero && num == 0 {
		r.AddError(field, "cannot be zero")
		return
	}

	// Check minimum value if set
	if validation.Min != nil && num < *validation.Min {
		r.AddError(field, fmt.Sprintf("must be greater than or equal to %v", *validation.Min))
		return
	}

	// Check maximum value if set
	if validation.Max != nil && num > *validation.Max {
		r.AddError(field, fmt.Sprintf("must be less than or equal to %v", *validation.Max))
		return
	}
}

// Integer validates if a string value is a valid integer
func (r *Rules) Integer(value string, field string, opts ...NumericValidation) {
	if value == "" {
		return // Skip empty values unless Required is also used
	}

	// First validate as number with decimals disabled
	validation := DefaultNumericValidation()
	if len(opts) > 0 {
		validation = opts[0]
	}
	validation.AllowDecimals = false

	r.Number(value, field, validation)
}

// Time validates if a string value is a valid time
func (r *Rules) Time(value string, field string) {
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		r.AddError(field, "must be a valid time")
	}
}

// Future validates if a time is in the future
func (r *Rules) Future(value time.Time, field string) {
	if value.Before(clock.Now()) {
		r.AddError(field, "must be a future date")
	}
}

// Past validates if a time is in the past
func (r *Rules) Past(value time.Time, field string) {
	if value.After(clock.Now()) {
		r.AddError(field, "must be a past date")
	}
}

// Range validates if a number is within a range
func (r *Rules) Range(value int, min int, max int, field string) {
	if value < min || value > max {
		r.AddError(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
}

// OneOf validates if a string is one of the allowed values
func (r *Rules) OneOf(value string, allowed []string, field string) {
	for _, v := range allowed {
		if value == v {
			return
		}
	}
	r.AddError(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
}

// In validates if a string is one of the allowed values
func (r *Rules) In(value string, allowed []string, field string) {
	for _, v := range allowed {
		if value == v {
			return
		}
	}
	r.AddError(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
}

// Matches validates if a string matches a regular expression
func (r *Rules) Matches(value string, pattern string, field string) {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil || !matched {
		r.AddError(field, "has invalid format")
	}
}

// Slug validates if a string is a valid slug
func (r *Rules) Slug(value string, field string) {
	if !slug.IsValidSlug(value) {
		r.AddError(field, "must be a valid slug (lowercase letters, numbers, and hyphens only)")
	}
}

// Custom allows for custom validation rules
func (r *Rules) Custom(valid bool, field string, message string) {
	if !valid {
		r.AddError(field, message)
	}
}

// AddError adds an error for a field
func (r *Rules) AddError(field, message string) {
	r.errors[field] = message
}

// HasErrors checks if there are any validation errors
func (r *Rules) HasErrors() bool {
	return len(r.errors) > 0
}

// GetErrors returns all validation errors
func (r *Rules) GetErrors() map[string]string {
	return r.errors
}

// ValidationResult returns validation errors if any exist
func (r *Rules) ValidationResult() error {
	if !r.HasErrors() {
		return nil
	}
	return &ValidationError{Errors: r.errors}
}
