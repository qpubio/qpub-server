package telemetry

import "github.com/qpubio/qpub-server/internal/shared/id"

type CounterName string

const (
	CounterEnqueued  CounterName = "job:enq"
	CounterCompleted CounterName = "job:ok"
	CounterFailed    CounterName = "job:fail"
	CounterRetried   CounterName = "job:retry"
	CounterDLQ       CounterName = "job:dlq"
	CounterDuration  CounterName = "job:duration_ms"
	GaugeDepth       CounterName = "queue:depth"
	GaugeWorkers     CounterName = "worker:active"
)

type Repository interface {
	Increment(projectID id.Int, name CounterName, delta int64)
	SetGauge(projectID id.Int, name CounterName, value int64)
	GetSnapshot(projectID id.Int) map[CounterName]int64
	Reset(projectID id.Int)
}

type Service interface {
	RecordEnqueue(projectID id.Int, payloadBytes int64)
	RecordCompleted(projectID id.Int, durationMs int64)
	RecordFailed(projectID id.Int)
	RecordRetried(projectID id.Int)
	RecordDLQ(projectID id.Int)
	SetQueueDepth(projectID id.Int, depth int64)
	SetActiveWorkers(projectID id.Int, count int64)
}
