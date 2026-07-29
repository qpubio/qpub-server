package snapshot

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/domain/project/stat/realtime"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"time"
)

// Service periodically snapshots in-memory telemetry to Redis.
type Service struct {
	logger           logger.Service
	instanceID       id.ULID
	subscriptionRepo subscription.Repository
	channelRepo      channel.Repository
	connectionRepo   connection.Repository
	telemetryRepo    domainTelemetry.Repository
	realtimeService  realtime.Service
	ticker           *time.Ticker
	stopChan         chan struct{}
}

// NewService creates a telemetry snapshot service.
func NewService(
	logger logger.Service,
	instanceID id.ULID,
	subscriptionRepo subscription.Repository,
	channelRepo channel.Repository,
	connectionRepo connection.Repository,
	telemetryRepo domainTelemetry.Repository,
	realtimeService realtime.Service,
) *Service {
	return &Service{
		logger:           logger,
		instanceID:       instanceID,
		subscriptionRepo: subscriptionRepo,
		channelRepo:      channelRepo,
		connectionRepo:   connectionRepo,
		telemetryRepo:    telemetryRepo,
		realtimeService:  realtimeService,
		stopChan:         make(chan struct{}),
	}
}

// Start begins the snapshot ticker.
func (s *Service) Start() {
	s.ticker = time.NewTicker(500 * time.Millisecond)
	go s.syncLoop()
	s.logger.Info(log.Stats, "Telemetry snapshot service started interval=%s", "500ms")
}

// Stop gracefully stops the snapshot service.
func (s *Service) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
	s.logger.Info(log.Stats, "Telemetry snapshot service stopped")
}

func (s *Service) syncLoop() {
	for {
		select {
		case <-s.ticker.C:
			s.syncStatsToRedis()
		case <-s.stopChan:
			return
		}
	}
}

type projectSnapshot struct {
	ProjectID         id.Int
	ConnectionCount   int
	ChannelCount      int
	SubscriberCount   int
	InboundCount      int64
	OutboundCount     int64
	OutboundDropped   int64
	BandwidthInbound  int64
	BandwidthOutbound int64
}

func (s *Service) syncStatsToRedis() {
	channels, err := s.channelRepo.FindAllLocal()
	if err != nil {
		s.logger.Error(log.Stats, "Failed to get channels for telemetry snapshot: %v", err)
		return
	}

	projectStats := make(map[id.Int]*projectSnapshot)

	for _, ch := range channels {
		projID := ch.ProjectID()
		s.ensureProjectStats(projectStats, projID)

		subs, err := s.subscriptionRepo.GetAllLocalForChannel(ch.FullName())
		if err != nil {
			continue
		}

		if len(subs) > 0 {
			projectStats[projID].ChannelCount++
			projectStats[projID].SubscriberCount += len(subs)
		}
	}

	counters, err := s.telemetryRepo.GetAllForInstance(s.instanceID)
	if err != nil {
		s.logger.Warn(log.Stats, "Failed to get telemetry counters for snapshot: %v", err)
	} else {
		for _, counter := range counters {
			s.checkAndResetAccumulatingStats(counter.ProjectID)

			freshCounter, err := s.telemetryRepo.GetCounter(counter.ProjectID, s.instanceID)
			if err != nil {
				continue
			}

			s.ensureProjectStats(projectStats, freshCounter.ProjectID)
			projectStats[freshCounter.ProjectID].InboundCount = freshCounter.InboundCount
			projectStats[freshCounter.ProjectID].OutboundCount = freshCounter.OutboundCount
			projectStats[freshCounter.ProjectID].OutboundDropped = freshCounter.OutboundDropped
			projectStats[freshCounter.ProjectID].BandwidthInbound = freshCounter.BandwidthInbound
			projectStats[freshCounter.ProjectID].BandwidthOutbound = freshCounter.BandwidthOutbound
		}
	}

	for projID := range projectStats {
		projectStats[projID].ConnectionCount = s.connectionRepo.CountByProject(projID)
	}

	for _, snapshot := range projectStats {
		s.writeSnapshotToRedis(snapshot)
	}
}

func (s *Service) ensureProjectStats(projectStats map[id.Int]*projectSnapshot, projID id.Int) {
	if _, exists := projectStats[projID]; !exists {
		projectStats[projID] = &projectSnapshot{ProjectID: projID}
	}
}

func (s *Service) checkAndResetAccumulatingStats(projectID id.Int) {
	key := realtime.NewKey(realtime.KeyMessageInbound, s.instanceID, projectID)
	_, err := s.realtimeService.Get(*key)
	if err != nil {
		s.logger.Debug(log.Stats, "Accumulating telemetry keys deleted - resetting in-memory counters projectID=%d instanceID=%s", projectID, s.instanceID)

		if err := s.telemetryRepo.ResetForProject(projectID); err != nil {
			s.logger.Warn(log.Stats, "Failed to reset in-memory telemetry counters projectID=%d: %v", projectID, err)
		}

		s.writeAccumulatingStatsToRedis(projectID, 0, 0, 0, 0, 0)
	}
}

func (s *Service) writeAccumulatingStatsToRedis(
	projectID id.Int,
	inboundCount, outboundCount, outboundDropped, bandwidthInbound, bandwidthOutbound int64,
) {
	keys := []struct {
		keyType realtime.KeyType
		value   int64
	}{
		{realtime.KeyMessageInbound, inboundCount},
		{realtime.KeyMessageOutbound, outboundCount},
		{realtime.KeyMessageDropped, outboundDropped},
		{realtime.KeyBandwidthInbound, bandwidthInbound},
		{realtime.KeyBandwidthOutbound, bandwidthOutbound},
	}

	for _, item := range keys {
		key := realtime.NewKey(item.keyType, s.instanceID, projectID)
		if err := s.realtimeService.Set(*key, item.value); err != nil {
			s.logger.Warn(log.Stats, "Failed to write telemetry stat to Redis projectID=%d keyType=%s: %v", projectID, item.keyType, err)
		}
	}
}

func (s *Service) writeSnapshotToRedis(snapshot *projectSnapshot) {
	gauges := []struct {
		keyType realtime.KeyType
		value   int64
	}{
		{realtime.KeyConnection, int64(snapshot.ConnectionCount)},
		{realtime.KeyChannel, int64(snapshot.ChannelCount)},
		{realtime.KeySubscriber, int64(snapshot.SubscriberCount)},
	}

	for _, gauge := range gauges {
		key := realtime.NewKey(gauge.keyType, s.instanceID, snapshot.ProjectID)
		if err := s.realtimeService.Set(*key, gauge.value); err != nil {
			s.logger.Warn(log.Stats, "Failed to sync gauge stat to Redis projectID=%d keyType=%s: %v", snapshot.ProjectID, gauge.keyType, err)
		}
	}

	s.writeAccumulatingStatsToRedis(
		snapshot.ProjectID,
		snapshot.InboundCount,
		snapshot.OutboundCount,
		snapshot.OutboundDropped,
		snapshot.BandwidthInbound,
		snapshot.BandwidthOutbound,
	)
}

// SyncStatsToRedis exposes snapshot sync for tests.
func (s *Service) SyncStatsToRedis() {
	s.syncStatsToRedis()
}
