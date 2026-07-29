package instance

// Service defines the instance service operations
type Service interface {
	// Register creates or updates a server instance and starts tracking it
	Register() error

	// Heartbeat updates the heartbeat timestamp for the current instance
	Heartbeat() error

	// Deregister marks an instance as inactive
	Deregister() error

	// CleanupStaleInstances finds and deactivates stale instances
	// Returns the number of instances marked as inactive
	CleanupStaleInstances() (int, error)

	// Get returns the current instance
	Get() (ServerInstance, error)
}
