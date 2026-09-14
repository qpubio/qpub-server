package tenant

// LifecycleStatus represents tenant provisioning state.
type LifecycleStatus string

const (
	StatusActive   LifecycleStatus = "active"
	StatusDeleting LifecycleStatus = "deleting"
)

func (s LifecycleStatus) IsActive() bool {
	return s == "" || s == StatusActive
}
