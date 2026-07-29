package telemetry

import (
	"fmt"

	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

// Service implements domain telemetry recording.
type Service struct {
	logger     logger.Service
	instanceID id.ULID
	repository domainTelemetry.Repository
}

// NewService creates a telemetry application service.
func NewService(
	logger logger.Service,
	instanceID id.ULID,
	repository domainTelemetry.Repository,
) domainTelemetry.Service {
	return &Service{
		logger:     logger,
		instanceID: instanceID,
		repository: repository,
	}
}

// Record applies a telemetry event to in-memory counters.
func (s *Service) Record(evt *domainTelemetry.Event) error {
	if evt == nil {
		return nil
	}

	instanceID := evt.InstanceID()
	if instanceID == "" {
		instanceID = s.instanceID
	}

	switch evt.Type() {
	case domainTelemetry.TypeInboundAccepted:
		data, ok := evt.Data().(domainTelemetry.InboundAcceptedData)
		if !ok {
			return fmt.Errorf("invalid data for %s: %T", evt.Type(), evt.Data())
		}
		return s.repository.RecordInbound(evt.ProjectID(), instanceID, data.ByteSize)

	case domainTelemetry.TypeOutboundDelivered:
		data, ok := evt.Data().(domainTelemetry.OutboundDeliveredData)
		if !ok {
			return fmt.Errorf("invalid data for %s: %T", evt.Type(), evt.Data())
		}
		return s.repository.RecordOutboundDelivered(evt.ProjectID(), instanceID, data.ByteSize)

	case domainTelemetry.TypeOutboundDropped:
		data, _ := evt.Data().(domainTelemetry.OutboundDroppedData)
		return s.repository.RecordOutboundDropped(evt.ProjectID(), instanceID, data.Reason)

	case domainTelemetry.TypeInboundRejected:
		return s.repository.RecordInboundRejected(evt.ProjectID(), instanceID)

	case domainTelemetry.TypeOutboundQueued,
		domainTelemetry.TypeOutboundFailed:
		return nil

	default:
		s.logger.Debug(log.Stats, "Ignoring unhandled telemetry event type=%s eventID=%s", evt.Type(), evt.ID())
		return nil
	}
}

// GetSnapshots returns counter snapshots for an instance.
func (s *Service) GetSnapshots(instanceID id.ULID) ([]*domainTelemetry.Snapshot, error) {
	counters, err := s.repository.GetAllForInstance(instanceID)
	if err != nil {
		return nil, err
	}

	snapshots := make([]*domainTelemetry.Snapshot, 0, len(counters))
	for _, counter := range counters {
		snapshots = append(snapshots, &domainTelemetry.Snapshot{
			ProjectID:         counter.ProjectID,
			InstanceID:        counter.InstanceID,
			InboundCount:      counter.InboundCount,
			OutboundCount:     counter.OutboundCount,
			OutboundDropped:   counter.OutboundDropped,
			BandwidthInbound:  counter.BandwidthInbound,
			BandwidthOutbound: counter.BandwidthOutbound,
		})
	}

	return snapshots, nil
}

// ResetForProject clears accumulating counters for a project.
func (s *Service) ResetForProject(projectID id.Int) error {
	return s.repository.ResetForProject(projectID)
}
