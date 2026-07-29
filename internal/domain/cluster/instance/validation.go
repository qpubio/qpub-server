package instance

import "github.com/qpubio/qpub-server/internal/shared/validation"

// Validator handles all instance-related validations
type Validator struct {
	*validation.Rules
}

func NewValidator() *Validator {
	return &Validator{
		Rules: validation.NewRules(),
	}
}

func (v *Validator) ValidateCreate(params CreateParams) error {
	// Instance ID validations
	v.Required(params.InstanceID, "instanceID")

	// Cluster component validations
	v.Required(params.ClusterScope, "clusterScope")
	v.Required(params.ClusterEnv, "clusterEnv")
	v.Required(params.ClusterRegion, "clusterRegion")
	v.Required(params.ClusterZone, "clusterZone")

	// VM component validations
	v.Required(params.VMScope, "vmScope")
	v.Required(params.VMEnv, "vmEnv")
	v.Required(params.VMRegion, "vmRegion")
	v.Required(params.VMZone, "vmZone")
	v.Required(params.VMRole, "vmRole")
	v.Required(params.VMSeq, "vmSeq")

	return nil
}

func (v *Validator) ValidateUpdate(params UpdateParams) error {
	// Status validations (if provided)
	if params.Status != "" {
		v.In(string(params.Status), []string{
			string(StatusActive),
			string(StatusInactive),
		}, "status")
	}

	return nil
}
