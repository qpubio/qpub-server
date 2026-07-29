package telemetry

import (
	"sync"

	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/queue/telemetry"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

type repository struct {
	mu       sync.RWMutex
	counters map[id.Int]map[domainTelemetry.CounterName]int64
}

func NewRepository() domainTelemetry.Repository {
	return &repository{
		counters: make(map[id.Int]map[domainTelemetry.CounterName]int64),
	}
}

func (r *repository) Increment(projectID id.Int, name domainTelemetry.CounterName, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.counters[projectID] == nil {
		r.counters[projectID] = make(map[domainTelemetry.CounterName]int64)
	}
	r.counters[projectID][name] += delta
}

func (r *repository) SetGauge(projectID id.Int, name domainTelemetry.CounterName, value int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.counters[projectID] == nil {
		r.counters[projectID] = make(map[domainTelemetry.CounterName]int64)
	}
	r.counters[projectID][name] = value
}

func (r *repository) GetSnapshot(projectID id.Int) map[domainTelemetry.CounterName]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := make(map[domainTelemetry.CounterName]int64)
	for k, v := range r.counters[projectID] {
		snapshot[k] = v
	}
	return snapshot
}

func (r *repository) Reset(projectID id.Int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.counters, projectID)
}
