package token

import (
	"github.com/qpubio/qpub-server/internal/shared/validation"
	"strconv"
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
func (v *Validator) ValidateCreateRevoke(params CreateRevokeParams) error {
	// TokenID validation
	v.Required(params.TokenID, "tokenID")
	v.MinLength(params.TokenID, 1, "tokenID")
	v.MaxLength(params.TokenID, 22, "tokenID")

	// OwnerID validation
	v.Number(strconv.Itoa(params.OwnerID), "ownerID", validation.NumericValidation{
		AllowNegative: false,
		AllowZero:     false,
		AllowDecimals: false,
	})

	// OwnerType validation
	v.In(string(params.OwnerType), []string{
		string(TokenTypeAPIKey),
	}, "ownerType")

	// ExpiresAt validation
	v.Required(params.ExpiresAt, "expiresAt")
	v.Future(params.ExpiresAt, "expiresAt")

	return v.ValidationResult()
}
