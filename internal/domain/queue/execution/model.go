package execution

import "fmt"

// Profile defines how jobs in a queue are executed.
type Profile string

const (
	ProfileExternal Profile = "external"
	ProfileManaged  Profile = "managed"
)

// ValidateProfile checks whether a profile is supported in the current runtime.
func ValidateProfile(profile Profile) error {
	switch profile {
	case ProfileExternal:
		return nil
	case ProfileManaged:
		return ErrManagedNotImplemented
	default:
		return fmt.Errorf("unknown execution profile: %s", profile)
	}
}
