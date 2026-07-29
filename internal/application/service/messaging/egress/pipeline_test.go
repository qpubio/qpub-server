package egress_test

import (
	"sync"
	"testing"
	"time"

	telemetryApp "github.com/qpubio/qpub-server/internal/application/service/messaging/telemetry"
	egressApp "github.com/qpubio/qpub-server/internal/application/service/messaging/egress"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	telemetryRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/stretchr/testify/require"
)

type nopLogger struct{}

func (nopLogger) Debug(log.Component, string, ...interface{}) {}
func (nopLogger) Info(log.Component, string, ...interface{})  {}
func (nopLogger) Warn(log.Component, string, ...interface{})  {}
func (nopLogger) Error(log.Component, string, ...interface{}) {}
func (nopLogger) Fatal(log.Component, string, ...interface{}) {}

type recordingConnection struct {
	mu          sync.Mutex
	messages    [][]byte
	block       chan struct{}
	sendStarted chan struct{}
	sendOnce    sync.Once
}

func (s *recordingConnection) Send(_ id.ULID, message []byte) error {
	s.sendOnce.Do(func() {
		if s.sendStarted != nil {
			close(s.sendStarted)
		}
	})
	if s.block != nil {
		<-s.block
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, append([]byte(nil), message...))
	return nil
}

func (s *recordingConnection) messagesCopy() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]byte, len(s.messages))
	for i, msg := range s.messages {
		out[i] = append([]byte(nil), msg...)
	}
	return out
}

func (s *recordingConnection) Register(_ *connection.Connection, _ func([]byte) error) error {
	return nil
}
func (s *recordingConnection) Unregister(_ id.ULID) error { return nil }
func (s *recordingConnection) Broadcast(_ id.Int, _ []byte) error {
	return nil
}
func (s *recordingConnection) Close(_ id.ULID) error { return nil }
func (s *recordingConnection) CloseAllByProject(_ id.Int) error {
	return nil
}
func (s *recordingConnection) CleanStaleConnections() (int, error) { return 0, nil }
func (s *recordingConnection) Get(_ id.ULID) (*connection.Connection, error) {
	return nil, connection.ErrConnectionNotFound
}

func TestPipeline_DeliversAndRecordsTelemetry(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	projectID := id.Int(9)
	connID := id.NewULID()
	repo := telemetryRepo.NewRepository(nopLogger{})
	telemetrySvc := telemetryApp.NewService(nopLogger{}, instanceID, repo)
	connSvc := &recordingConnection{}

	pipeline := egressApp.NewPipeline(egressApp.PipelineParams{
		ConnID:            connID,
		ProjectID:         projectID,
		InstanceID:        instanceID,
		ConnectionService: connSvc,
		TelemetryService:  telemetrySvc,
		Logger:            nopLogger{},
		Capacity:          4,
	})
	pipeline.Start()
	defer pipeline.Stop()

	require.NoError(t, pipeline.Enqueue([]byte("hello")))

	require.Eventually(t, func() bool {
		return len(connSvc.messagesCopy()) == 1
	}, time.Second, 10*time.Millisecond)

	counter, err := repo.GetCounter(projectID, instanceID)
	require.NoError(t, err)
	require.Equal(t, int64(1), counter.OutboundCount)
	require.Equal(t, int64(5), counter.BandwidthOutbound)
}

func TestPipeline_DropNewestRecordsTelemetry(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	projectID := id.Int(11)
	connID := id.NewULID()
	repo := telemetryRepo.NewRepository(nopLogger{})
	telemetrySvc := telemetryApp.NewService(nopLogger{}, instanceID, repo)

	connSvc := &recordingConnection{
		block:       make(chan struct{}),
		sendStarted: make(chan struct{}),
	}

	pipeline := egressApp.NewPipeline(egressApp.PipelineParams{
		ConnID:            connID,
		ProjectID:         projectID,
		InstanceID:        instanceID,
		ConnectionService: connSvc,
		TelemetryService:  telemetrySvc,
		Logger:            nopLogger{},
		Capacity:          1,
	})
	pipeline.Start()

	require.NoError(t, pipeline.Enqueue([]byte("first")))
	<-connSvc.sendStarted
	require.NoError(t, pipeline.Enqueue([]byte("second")))
	require.NoError(t, pipeline.Enqueue([]byte("third")))

	counter, err := repo.GetCounter(projectID, instanceID)
	require.NoError(t, err)
	require.Equal(t, int64(1), counter.OutboundDropped)

	close(connSvc.block)
	pipeline.Stop()
}
