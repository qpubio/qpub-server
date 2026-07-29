package session_test

import (
	"testing"
	"time"

	sessionApp "github.com/qpubio/qpub-server/internal/application/service/messaging/session"
	telemetryApp "github.com/qpubio/qpub-server/internal/application/service/messaging/telemetry"
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

type stubConnection struct {
	sent [][]byte
}

func (s *stubConnection) Register(_ *connection.Connection, _ func([]byte) error) error {
	return nil
}
func (s *stubConnection) Unregister(_ id.ULID) error { return nil }
func (s *stubConnection) Send(_ id.ULID, message []byte) error {
	s.sent = append(s.sent, append([]byte(nil), message...))
	return nil
}
func (s *stubConnection) Broadcast(_ id.Int, _ []byte) error { return nil }
func (s *stubConnection) Close(_ id.ULID) error              { return nil }
func (s *stubConnection) CloseAllByProject(_ id.Int) error   { return nil }
func (s *stubConnection) CleanStaleConnections() (int, error) {
	return 0, nil
}
func (s *stubConnection) Get(_ id.ULID) (*connection.Connection, error) {
	return nil, connection.ErrConnectionNotFound
}

func TestService_GetOrCreateSubscriptionReusesActiveSubscription(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	repo := telemetryRepo.NewRepository(nopLogger{})
	telemetrySvc := telemetryApp.NewService(nopLogger{}, instanceID, repo)
	connSvc := &stubConnection{}
	svc := sessionApp.NewService(nopLogger{}, instanceID, connSvc, telemetrySvc)

	clientID := id.NewULID()
	first := svc.GetOrCreateSubscription(clientID)
	second := svc.GetOrCreateSubscription(clientID)

	require.Equal(t, first.ID(), second.ID())
}

func TestService_DeliverRoutesThroughEgress(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	projectID := id.Int(3)
	repo := telemetryRepo.NewRepository(nopLogger{})
	telemetrySvc := telemetryApp.NewService(nopLogger{}, instanceID, repo)
	connSvc := &stubConnection{}
	svc := sessionApp.NewService(nopLogger{}, instanceID, connSvc, telemetrySvc)

	connID := id.NewULID()
	clientID := id.NewULID()
	svc.RegisterConnection(connID, projectID, clientID)

	require.NoError(t, svc.Deliver(clientID, []byte("payload")))

	require.Eventually(t, func() bool {
		return len(connSvc.sent) == 1
	}, time.Second, 10*time.Millisecond)

	svc.UnregisterConnection(connID, clientID)

	require.Equal(t, []byte("payload"), connSvc.sent[0])
}
