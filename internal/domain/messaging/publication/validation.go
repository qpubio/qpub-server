package publication

import (
	"github.com/qpubio/qpub-server/internal/shared/validation"
	"strconv"
)

// Validator handles all publisher validations
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

	// ChannelName validation
	v.Required(params.ChannelName, "channelName")
	v.MinLength(params.ChannelName, 1, "channelName")
	v.MaxLength(params.ChannelName, 100, "channelName")

	return v.ValidationResult()
}
