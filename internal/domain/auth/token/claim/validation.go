package claim

import (
	"github.com/qpubio/qpub-server/internal/shared/validation"
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

// ValidateCreateAPIKeyClaim validates the CreateAPIKeyClaimParams
func (v *Validator) ValidateCreateAPIKeyClaim(params CreateAPIKeyClaimParams) error {
	// Alias validation - only validate if provided
	if params.Alias != "" {
		v.MinLength(params.Alias, 1, "alias")
		v.MaxLength(params.Alias, 120, "alias")
	}

	// Permission validation - only validate if provided
	if len(params.Permission) > 0 {
		v.JSON(params.Permission, "permission")
	}

	return v.ValidationResult()
}
