package schedule

import (
	"context"
	"time"
)

// Entry represents a recurring schedule for a queue job.
type Entry struct {
	ID          int64
	ProjectID   int64
	QueueName   string
	CronSpec    string
	Payload     []byte
	LockTimeout time.Duration
	Enabled     bool
}

// Repository persists schedule entries.
type Repository interface {
	ListEnabled() ([]Entry, error)
	Upsert(entry *Entry) error
	Delete(id int64) error
}

// Service manages cron-based job scheduling.
type Service interface {
	Register(ctx context.Context, entry Entry) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsLeader() bool
}

// Handler processes scheduled job payloads.
type Handler func(ctx context.Context, payload []byte) error
