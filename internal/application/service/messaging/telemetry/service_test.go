package telemetry_test

import (
	"testing"

	telemetryApp "github.com/qpubio/qpub-server/internal/application/service/messaging/telemetry"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
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

func TestRecord_InboundAcceptedIncrementsOnce(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	repo := telemetryRepo.NewRepository(nopLogger{})
	svc := telemetryApp.NewService(nopLogger{}, instanceID, repo)

	evt := domainTelemetry.NewEvent(
		domainTelemetry.TypeInboundAccepted,
		id.Int(7),
		instanceID,
		domainTelemetry.InboundAcceptedData{
			Channel:  "room",
			ByteSize: 128,
			Source:   "websocket",
		},
	)
	require.NoError(t, svc.Record(evt))
	require.NoError(t, svc.Record(evt))

	counter, err := repo.GetCounter(id.Int(7), instanceID)
	require.NoError(t, err)
	require.Equal(t, int64(2), counter.InboundCount)
	require.Equal(t, int64(256), counter.BandwidthInbound)
	require.Equal(t, int64(0), counter.OutboundCount)
}

func TestRecord_OutboundDeliveredIncrementsOnce(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	repo := telemetryRepo.NewRepository(nopLogger{})
	svc := telemetryApp.NewService(nopLogger{}, instanceID, repo)

	evt := domainTelemetry.NewEvent(
		domainTelemetry.TypeOutboundDelivered,
		id.Int(7),
		instanceID,
		domainTelemetry.OutboundDeliveredData{
			ByteSize: 64,
		},
	)
	require.NoError(t, svc.Record(evt))

	counter, err := repo.GetCounter(id.Int(7), instanceID)
	require.NoError(t, err)
	require.Equal(t, int64(1), counter.OutboundCount)
	require.Equal(t, int64(64), counter.BandwidthOutbound)
}

func TestRecord_OutboundDroppedIncrementsOnce(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	repo := telemetryRepo.NewRepository(nopLogger{})
	svc := telemetryApp.NewService(nopLogger{}, instanceID, repo)

	evt := domainTelemetry.NewEvent(
		domainTelemetry.TypeOutboundDropped,
		id.Int(7),
		instanceID,
		domainTelemetry.OutboundDroppedData{},
	)
	require.NoError(t, svc.Record(evt))

	counter, err := repo.GetCounter(id.Int(7), instanceID)
	require.NoError(t, err)
	require.Equal(t, int64(1), counter.OutboundDropped)
}
