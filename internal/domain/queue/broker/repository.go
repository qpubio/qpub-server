package broker

import (
	"context"
	"time"
)

// Message represents a broker message for a job.
type Message struct {
	Subject  string
	Data     []byte
	Sequence uint64
}

// MessageHandler handles incoming broker messages.
type MessageHandler func(msg Message) error

// Repository defines the durable job transport port.
type Repository interface {
	EnsureStream(subject string, retention time.Duration) error
	Publish(ctx context.Context, subject string, data []byte) (uint64, error)
	Pull(ctx context.Context, subject string, batchSize int, wait time.Duration) ([]Message, error)
	Ack(ctx context.Context, subject string, sequence uint64) error
	Nack(ctx context.Context, subject string, sequence uint64, delay time.Duration) error
	PublishDLQ(ctx context.Context, subject string, data []byte) error
	Shutdown(ctx context.Context) error
}
