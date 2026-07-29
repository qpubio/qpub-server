package snapshot_test

import (
	"testing"

	shared "github.com/qpubio/qpub-server/internal/config/shared"
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/domain/project/stat/realtime"
	telemetrySnapshot "github.com/qpubio/qpub-server/internal/application/service/messaging/telemetry/snapshot"
	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	channelRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/channel"
	connectionRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/connection"
	telemetryRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/telemetry"
	subscriptionRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLogger struct{}

func (mockLogger) Info(log.Component, string, ...interface{})  {}
func (mockLogger) Debug(log.Component, string, ...interface{}) {}
func (mockLogger) Warn(log.Component, string, ...interface{})  {}
func (mockLogger) Error(log.Component, string, ...interface{}) {}
func (mockLogger) Fatal(log.Component, string, ...interface{}) {}

type mockRealtimeService struct {
	stats map[string]int64
}

func newMockRealtimeService() *mockRealtimeService {
	return &mockRealtimeService{stats: make(map[string]int64)}
}

func (m *mockRealtimeService) Incr(key realtime.Key) error {
	m.stats[key.String()]++
	return nil
}
func (m *mockRealtimeService) IncrBy(key realtime.Key, value int64) error {
	m.stats[key.String()] += value
	return nil
}
func (m *mockRealtimeService) Decr(key realtime.Key) error {
	if m.stats[key.String()] > 0 {
		m.stats[key.String()]--
	}
	return nil
}
func (m *mockRealtimeService) DecrBy(key realtime.Key, value int64) error {
	m.stats[key.String()] -= value
	return nil
}
func (m *mockRealtimeService) Get(key realtime.Key) (int64, error) {
	return m.stats[key.String()], nil
}
func (m *mockRealtimeService) Set(key realtime.Key, value int64) error {
	m.stats[key.String()] = value
	return nil
}
func (m *mockRealtimeService) GetByPattern(string) (map[string]int64, error) { return m.stats, nil }
func (m *mockRealtimeService) Reset(key realtime.Key) error {
	delete(m.stats, key.String())
	return nil
}
func (m *mockRealtimeService) ResetByPattern(string) error {
	m.stats = make(map[string]int64)
	return nil
}
func (m *mockRealtimeService) GetSummary(id.Int) (map[string]int64, error) { return m.stats, nil }

func initIDService(t *testing.T) {
	t.Helper()
	clock.Init()
	cfg := &shared.ID{
		HashSalt:   "test-salt-for-testing-12345",
		HashLength: 8,
		ULIDLength: 22,
	}
	_ = id.Init(cfg)
}

func TestService_WritesTelemetryKeysToRedis(t *testing.T) {
	initIDService(t)

	instanceID := id.NewULID()
	projectID := id.Int(123)
	mockLog := mockLogger{}
	mockRealtime := newMockRealtimeService()

	chanRepo := channelRepo.NewRepository(mockLog)
	subRepo := subscriptionRepo.NewRepository(mockLog, instanceID)
	connRepo := connectionRepo.NewRepository(mockLog)
	telemetryRepository := telemetryRepo.NewRepository(mockLog)

	require.NoError(t, telemetryRepository.RecordInbound(projectID, instanceID, 50))
	require.NoError(t, telemetryRepository.RecordOutboundDelivered(projectID, instanceID, 40))
	require.NoError(t, telemetryRepository.RecordOutboundDropped(projectID, instanceID, backpressure.DropReasonWriteTimeout))

	ch := channel.New(channel.NewName("room", projectID), instanceID)
	require.NoError(t, chanRepo.Create(ch))
	sub := subscription.New(id.NewULID(), 10)
	require.NoError(t, subRepo.AddToChannel(ch.FullName(), sub))
	ch.IncrementLocalSubscriptions()

	snapshotSvc := telemetrySnapshot.NewService(
		mockLog,
		instanceID,
		subRepo,
		chanRepo,
		connRepo,
		telemetryRepository,
		mockRealtime,
	)
	snapshotSvc.SyncStatsToRedis()

	inboundKey := realtime.NewKey(realtime.KeyMessageInbound, instanceID, projectID)
	outboundKey := realtime.NewKey(realtime.KeyMessageOutbound, instanceID, projectID)
	droppedKey := realtime.NewKey(realtime.KeyMessageDropped, instanceID, projectID)
	bwInKey := realtime.NewKey(realtime.KeyBandwidthInbound, instanceID, projectID)
	bwOutKey := realtime.NewKey(realtime.KeyBandwidthOutbound, instanceID, projectID)

	inbound, err := mockRealtime.Get(*inboundKey)
	require.NoError(t, err)
	assert.Equal(t, int64(1), inbound)

	outbound, err := mockRealtime.Get(*outboundKey)
	require.NoError(t, err)
	assert.Equal(t, int64(1), outbound)

	dropped, err := mockRealtime.Get(*droppedKey)
	require.NoError(t, err)
	assert.Equal(t, int64(1), dropped)

	bwIn, err := mockRealtime.Get(*bwInKey)
	require.NoError(t, err)
	assert.Equal(t, int64(50), bwIn)

	bwOut, err := mockRealtime.Get(*bwOutKey)
	require.NoError(t, err)
	assert.Equal(t, int64(40), bwOut)
}
