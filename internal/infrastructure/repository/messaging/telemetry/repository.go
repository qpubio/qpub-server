package telemetry

import (
	"fmt"
	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
)

type repository struct {
	logger   logger.Service
	counters map[string]*telemetry.Counter
	mu       sync.RWMutex
}

// NewRepository creates an in-memory telemetry counter repository.
func NewRepository(logger logger.Service) telemetry.Repository {
	return &repository{
		logger:   logger,
		counters: make(map[string]*telemetry.Counter),
	}
}

func (r *repository) key(projectID id.Int, instanceID id.ULID) string {
	return fmt.Sprintf("%d:%s", projectID, instanceID)
}

func (r *repository) getOrCreate(projectID id.Int, instanceID id.ULID) *telemetry.Counter {
	key := r.key(projectID, instanceID)

	r.mu.RLock()
	counter, exists := r.counters[key]
	r.mu.RUnlock()
	if exists {
		return counter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	counter, exists = r.counters[key]
	if exists {
		return counter
	}

	counter = telemetry.NewCounter(projectID, instanceID)
	r.counters[key] = counter
	return counter
}

func (r *repository) RecordInbound(projectID id.Int, instanceID id.ULID, byteSize int64) error {
	counter := r.getOrCreate(projectID, instanceID)

	r.mu.Lock()
	counter.InboundCount++
	if byteSize > 0 {
		counter.BandwidthInbound += byteSize
	}
	r.mu.Unlock()

	return nil
}

func (r *repository) RecordOutboundDelivered(projectID id.Int, instanceID id.ULID, byteSize int64) error {
	counter := r.getOrCreate(projectID, instanceID)

	r.mu.Lock()
	counter.OutboundCount++
	if byteSize > 0 {
		counter.BandwidthOutbound += byteSize
	}
	r.mu.Unlock()

	return nil
}

func (r *repository) RecordOutboundDropped(projectID id.Int, instanceID id.ULID, reason backpressure.DropReason) error {
	counter := r.getOrCreate(projectID, instanceID)

	r.mu.Lock()
	counter.OutboundDropped++
	if reason != "" {
		if counter.DroppedByReason == nil {
			counter.DroppedByReason = make(map[backpressure.DropReason]int64)
		}
		counter.DroppedByReason[reason]++
	}
	r.mu.Unlock()

	return nil
}

func (r *repository) RecordInboundRejected(projectID id.Int, instanceID id.ULID) error {
	counter := r.getOrCreate(projectID, instanceID)

	r.mu.Lock()
	counter.InboundRejected++
	r.mu.Unlock()

	return nil
}

func (r *repository) GetCounter(projectID id.Int, instanceID id.ULID) (*telemetry.Counter, error) {
	key := r.key(projectID, instanceID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	counter, exists := r.counters[key]
	if !exists {
		return telemetry.NewCounter(projectID, instanceID), nil
	}

	copy := *counter
	if counter.DroppedByReason != nil {
		copy.DroppedByReason = make(map[backpressure.DropReason]int64, len(counter.DroppedByReason))
		for reason, count := range counter.DroppedByReason {
			copy.DroppedByReason[reason] = count
		}
	}
	return &copy, nil
}

func (r *repository) GetAllForInstance(instanceID id.ULID) ([]*telemetry.Counter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*telemetry.Counter, 0)
	instanceIDStr := string(instanceID)

	for _, counter := range r.counters {
		if counter.InstanceID == instanceID || string(counter.InstanceID) == instanceIDStr {
			copy := *counter
			result = append(result, &copy)
		}
	}

	return result, nil
}

func (r *repository) ResetForProject(projectID id.Int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	resetCount := 0
	for _, counter := range r.counters {
		if counter.ProjectID == projectID {
			counter.InboundCount = 0
			counter.OutboundCount = 0
			counter.InboundRejected = 0
			counter.OutboundDropped = 0
			counter.DroppedByReason = make(map[backpressure.DropReason]int64)
			counter.BandwidthInbound = 0
			counter.BandwidthOutbound = 0
			resetCount++
		}
	}

	if resetCount > 0 {
		r.logger.Debug(log.Stats, `Reset telemetry counters for project projectID=%v count=%v`, projectID,
			resetCount)
	}

	return nil
}

func (r *repository) ResetForInstance(instanceID id.ULID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instanceIDStr := string(instanceID)
	keysToDelete := make([]string, 0)

	for key, counter := range r.counters {
		if counter.InstanceID == instanceID || string(counter.InstanceID) == instanceIDStr {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(r.counters, key)
	}

	return nil
}
