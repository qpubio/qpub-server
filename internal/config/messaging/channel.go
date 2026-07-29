package messaging

import (
	"github.com/qpubio/qpub-server/internal/shared/duration"
	"time"
)

// ChannelConfig contains configuration for messaging channels
type ChannelConfig struct {
	// CleanupDelay is the time to wait before deleting an empty channel
	// This prevents premature deletion when clients reconnect quickly
	CleanupDelay duration.DurationVar `env:"MESSAGING_CHANNEL_CLEANUP_DELAY" envDefault:"30s"`
}

// GetCleanupDelay returns the cleanup delay as a time.Duration
func (c *ChannelConfig) GetCleanupDelay() time.Duration {
	return c.CleanupDelay.Duration
}
