package validation

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error with multiple field errors
type ValidationError struct {
	Errors map[string]string
}

func (ve *ValidationError) Error() string {
	var errors []string
	for field, msg := range ve.Errors {
		errors = append(errors, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(errors, "; ")
}

// FieldError represents a single field validation error
type FieldError struct {
	Field   string
	Message string
}

func (fe *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", fe.Field, fe.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError() *ValidationError {
	return &ValidationError{
		Errors: make(map[string]string),
	}
}

// NewFieldError creates a new FieldError
func NewFieldError(field, message string) *FieldError {
	return &FieldError{
		Field:   field,
		Message: message,
	}
}
