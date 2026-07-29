package telemetry_test

import (
	"testing"

	"github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/stretchr/testify/require"
)

func TestEventTypeClassification(t *testing.T) {
	require.True(t, telemetry.TypeInboundAccepted.IsInbound())
	require.True(t, telemetry.TypeInboundRejected.IsInbound())
	require.False(t, telemetry.TypeOutboundDelivered.IsInbound())

	require.True(t, telemetry.TypeOutboundDelivered.IsOutbound())
	require.True(t, telemetry.TypeOutboundDropped.IsOutbound())
	require.False(t, telemetry.TypeInboundAccepted.IsOutbound())

	require.True(t, telemetry.TypeInboundAccepted.CountsTowardInbound())
	require.False(t, telemetry.TypeInboundRejected.CountsTowardInbound())

	require.True(t, telemetry.TypeOutboundDelivered.CountsTowardOutbound())
	require.True(t, telemetry.TypeOutboundDropped.CountsTowardDropped())
}

func TestNewEvent(t *testing.T) {
	testsupport.InitRuntime()
	data := telemetry.InboundAcceptedData{
		EnvelopeID: id.NewULID(),
		Channel:    "room",
		ByteSize:   128,
		Source:     "websocket",
	}
	evt := telemetry.NewEvent(
		telemetry.TypeInboundAccepted,
		id.Int(99),
		id.NewULID(),
		data,
	)

	require.NotEmpty(t, evt.ID())
	require.Equal(t, telemetry.TypeInboundAccepted, evt.Type())
	require.Equal(t, id.Int(99), evt.ProjectID())
	require.False(t, evt.Timestamp().IsZero())

	parsed, ok := evt.Data().(telemetry.InboundAcceptedData)
	require.True(t, ok)
	require.Equal(t, data.Channel, parsed.Channel)
	require.Equal(t, data.ByteSize, parsed.ByteSize)
}
