package telemetry

import (
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/queue/telemetry"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

type Service struct {
	repository domainTelemetry.Repository
}

func NewService(repository domainTelemetry.Repository) domainTelemetry.Service {
	return &Service{repository: repository}
}

func (s *Service) RecordEnqueue(projectID id.Int, payloadBytes int64) {
	s.repository.Increment(projectID, domainTelemetry.CounterEnqueued, 1)
}

func (s *Service) RecordCompleted(projectID id.Int, durationMs int64) {
	s.repository.Increment(projectID, domainTelemetry.CounterCompleted, 1)
	s.repository.Increment(projectID, domainTelemetry.CounterDuration, durationMs)
}

func (s *Service) RecordFailed(projectID id.Int) {
	s.repository.Increment(projectID, domainTelemetry.CounterFailed, 1)
}

func (s *Service) RecordRetried(projectID id.Int) {
	s.repository.Increment(projectID, domainTelemetry.CounterRetried, 1)
}

func (s *Service) RecordDLQ(projectID id.Int) {
	s.repository.Increment(projectID, domainTelemetry.CounterDLQ, 1)
}

func (s *Service) SetQueueDepth(projectID id.Int, depth int64) {
	s.repository.SetGauge(projectID, domainTelemetry.GaugeDepth, depth)
}

func (s *Service) SetActiveWorkers(projectID id.Int, count int64) {
	s.repository.SetGauge(projectID, domainTelemetry.GaugeWorkers, count)
}
