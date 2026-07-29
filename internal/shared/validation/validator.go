package validation

// Validator interface defines the contract for validatable entities
type Validator interface {
	Validate() error
}

// ValidatorFunc is a function type that implements Validator
type ValidatorFunc func() error

// Validate implements the Validator interface for ValidatorFunc
func (f ValidatorFunc) Validate() error {
	return f()
}

// Chain allows multiple validators to be chained together
type Chain struct {
	validators []Validator
}

// NewValidationChain creates a new validation chain
func NewValidationChain(validators ...Validator) *Chain {
	return &Chain{
		validators: validators,
	}
}

// Add adds a validator to the chain
func (c *Chain) Add(validator Validator) *Chain {
	c.validators = append(c.validators, validator)
	return c
}

// AddFunc adds a validator function to the chain
func (c *Chain) AddFunc(validatorFunc func() error) *Chain {
	c.validators = append(c.validators, ValidatorFunc(validatorFunc))
	return c
}

// Validate runs all validators in the chain
func (c *Chain) Validate() error {
	for _, validator := range c.validators {
		if err := validator.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ValidateAll runs all validators and collects all errors
func (c *Chain) ValidateAll() *ValidationError {
	validationError := NewValidationError()

	for _, validator := range c.validators {
		if err := validator.Validate(); err != nil {
			switch e := err.(type) {
			case *ValidationError:
				for field, msg := range e.Errors {
					validationError.Errors[field] = msg
				}
			case *FieldError:
				validationError.Errors[e.Field] = e.Message
			default:
				validationError.Errors["general"] = err.Error()
			}
		}
	}

	if len(validationError.Errors) == 0 {
		return nil
	}

	return validationError
}
