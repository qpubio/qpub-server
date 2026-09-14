package queue

// LifecycleStatus represents queue provisioning state.
type LifecycleStatus string

const (
	StatusActive   LifecycleStatus = "active"
	StatusDeleting LifecycleStatus = "deleting"
)

func (s LifecycleStatus) IsActive() bool {
	return s == "" || s == StatusActive
}
