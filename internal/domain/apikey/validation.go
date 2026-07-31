package apikey

import (
	"github.com/qpubio/qpub-server/internal/shared/validation"
	"strconv"
	"time"
)

type Validator struct {
	*validation.Rules
}

// NewValidator creates a new Validator instance
func NewValidator() *Validator {
	return &Validator{
		Rules: validation.NewRules(),
	}
}

// ValidateCreate validates the CreateParams
func (v *Validator) ValidateCreate(params CreateParams) error {
	// ProjectID validation
	v.Number(strconv.Itoa(params.ProjectID), "projectID", validation.NumericValidation{
		AllowNegative: false,
		AllowZero:     false,
		AllowDecimals: false,
	})

	// Name validation
	v.Required(params.Name, "name")
	v.MinLength(params.Name, 2, "name")
	v.MaxLength(params.Name, 100, "name")

	// Permissions validation
	if params.Permission != nil {
		v.JSON(params.Permission, "permission")
	}

	// Metadata validation
	if params.Metadata != nil {
		v.JSON(params.Metadata, "metadata")
	}

	// Status validation
	if params.Status != "" {
		v.In(string(params.Status), []string{
			string(StatusActive),
			string(StatusInactive),
		}, "status")
	}

	// ExpiresAt validation
	if params.ExpiresAt != nil {
		v.Time(params.ExpiresAt.Format(time.RFC3339), "expiresAt")
		v.Future(*params.ExpiresAt, "expiresAt")
	}

	return v.ValidationResult()
}

// ValidateUpdate validates the UpdateParams
func (v *Validator) ValidateUpdate(params UpdateParams) error {
	// Name validation
	v.Required(params.Name, "name")
	v.MinLength(params.Name, 2, "name")
	v.MaxLength(params.Name, 100, "name")

	// Permission validation (optional on update — nil keeps existing)
	if params.Permission != nil {
		v.JSON(params.Permission, "permission")
	}

	// Metadata validation
	if params.Metadata != nil {
		v.JSON(params.Metadata, "metadata")
	}

	// Status validation
	if params.Status != "" {
		v.In(string(params.Status), []string{
			string(StatusActive),
			string(StatusInactive),
		}, "status")
	}

	// LastUsedAt validation
	if params.LastUsedAt != nil {
		v.Time(params.LastUsedAt.Format(time.RFC3339), "lastUsedAt")
		v.Past(*params.LastUsedAt, "lastUsedAt")
	}

	// ExpiresAt validation
	if params.ExpiresAt != nil {
		v.Time(params.ExpiresAt.Format(time.RFC3339), "expiresAt")
		v.Future(*params.ExpiresAt, "expiresAt")
	}

	return v.ValidationResult()
}
