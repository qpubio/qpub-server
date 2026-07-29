package envelope_test

import (
	"testing"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/messaging/envelope"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/stretchr/testify/require"
)

func TestNewEnvelope(t *testing.T) {
	testsupport.InitRuntime()
	t.Run("defaults publishedAt and copies payload", func(t *testing.T) {
		payload := []byte(`{"action":10}`)
		env := envelope.New(envelope.CreateParams{
			Direction: envelope.DirectionInbound,
			ProjectID: id.Int(42),
			Channel:   "chat.room",
			Payload:   payload,
			Source:    envelope.SourceWebSocket,
		})

		require.NotEmpty(t, env.ID())
		require.False(t, env.PublishedAt().IsZero())
		require.Equal(t, envelope.DirectionInbound, env.Direction())
		require.Equal(t, id.Int(42), env.ProjectID())
		require.Equal(t, "chat.room", env.Channel())
		require.Equal(t, envelope.SourceWebSocket, env.Source())
		require.Equal(t, int64(len(payload)), env.Size())
		require.True(t, env.IsInbound())
		require.False(t, env.IsOutbound())

		payload[0] = 'X'
		require.Equal(t, byte('{'), env.Payload()[0])
	})

	t.Run("respects explicit publishedAt", func(t *testing.T) {
		at := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
		env := envelope.New(envelope.CreateParams{
			Direction:   envelope.DirectionOutbound,
			ProjectID:   id.Int(1),
			Channel:     "events",
			Source:      envelope.SourceNATS,
			PublishedAt: at,
		})

		require.Equal(t, at, env.PublishedAt())
		require.True(t, env.IsOutbound())
	})

	t.Run("WithID preserves other fields", func(t *testing.T) {
		env := envelope.New(envelope.CreateParams{
			Direction: envelope.DirectionInbound,
			ProjectID: id.Int(7),
			Channel:   "a",
			Source:    envelope.SourceREST,
		})
		fixedID := id.NewULID()
		withID := env.WithID(fixedID)

		require.Equal(t, fixedID, withID.ID())
		require.Equal(t, env.Channel(), withID.Channel())
		require.Equal(t, env.Source(), withID.Source())
	})
}
