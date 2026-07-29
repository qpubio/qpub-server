package channel

// Repository defines the interface for channel persistence operations
type Repository interface {
	// Create persists a new channel to the storage
	Create(channel *Channel) error

	// Update updates an existing channel in the storage
	Update(channel *Channel) error

	// Delete deletes a channel from the storage
	Delete(channel *Channel) error

	// FindByName retrieves a channel by its full name
	FindByName(fullName string) (*Channel, error)

	// FindAllLocal retrieves all local channels in this instance
	FindAllLocal() ([]*Channel, error)
}
