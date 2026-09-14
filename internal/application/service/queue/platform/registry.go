package platform

import (
	"context"
	"fmt"
	"sync"
	"time"

	taskType "github.com/qpubio/qpub-server/internal/shared/type/task"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

const PlatformProjectID id.Int = 0

// Handler processes a platform job payload.
type Handler func(ctx context.Context, payload []byte) error

// IdempotencyBucket chooses the enqueue idempotency key time bucket.
type IdempotencyBucket int

const (
	BucketMinute IdempotencyBucket = iota
	BucketTenMinutes
	BucketHour
	BucketDay
)

// TaskDefinition describes a platform scheduled task.
type TaskDefinition struct {
	Name              taskType.TaskName
	Schedule          string
	LockTimeout       time.Duration
	IdempotencyBucket IdempotencyBucket
	Handler           Handler
}

// Registry holds platform task definitions.
type Registry struct {
	definitions map[taskType.TaskName]TaskDefinition
	mu          sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[taskType.TaskName]TaskDefinition),
	}
}

func (r *Registry) Register(def TaskDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.definitions[def.Name]; exists {
		return fmt.Errorf("platform task %s already registered", def.Name)
	}
	r.definitions[def.Name] = def
	return nil
}

func (r *Registry) All() []TaskDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]TaskDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		tasks = append(tasks, def)
	}
	return tasks
}

func (r *Registry) Get(name taskType.TaskName) (TaskDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.definitions[name]
	return def, ok
}

// QueueName returns the platform queue name for a task.
func QueueName(taskName taskType.TaskName) string {
	return fmt.Sprintf("_platform.%s", taskName)
}

// IdempotencyKey builds a per-tick enqueue key for the task.
func IdempotencyKey(name taskType.TaskName, bucket IdempotencyBucket, now time.Time) string {
	now = now.UTC()
	switch bucket {
	case BucketTenMinutes:
		t := now.Truncate(10 * time.Minute)
		return fmt.Sprintf("%s:%s", name, t.Format("200601021504"))
	case BucketHour:
		return fmt.Sprintf("%s:%s", name, now.Format("2006010215"))
	case BucketDay:
		return fmt.Sprintf("%s:%s", name, now.Format("20060102"))
	default:
		return fmt.Sprintf("%s:%s", name, now.Format("200601021504"))
	}
}
