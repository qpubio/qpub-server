package receipt_test

import (
	"testing"

	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	"github.com/qpubio/qpub-server/internal/domain/messaging/receipt"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/stretchr/testify/require"
)

func TestReceiptHelpers(t *testing.T) {
	testsupport.InitRuntime()
	envID := id.NewULID()

	t.Run("ingress ack", func(t *testing.T) {
		r := receipt.IngressAck(envID)
		require.True(t, r.IsAck())
		require.True(t, r.IsSuccess())
		require.Equal(t, receipt.StageIngress, r.Stage())
		require.Equal(t, receipt.OutcomeAccepted, r.Outcome())
	})

	t.Run("ingress nack", func(t *testing.T) {
		r := receipt.IngressNack(envID, "forbidden", protocol.ErrForbidden)
		require.True(t, r.IsNack())
		require.False(t, r.IsSuccess())
		require.Equal(t, protocol.ErrForbidden, r.ErrorCode())
	})

	t.Run("egress delivered", func(t *testing.T) {
		r := receipt.EgressDelivered(envID)
		require.True(t, r.IsSuccess())
		require.Equal(t, receipt.OutcomeDelivered, r.Outcome())
	})

	t.Run("egress dropped", func(t *testing.T) {
		r := receipt.EgressDropped(envID, "buffer full")
		require.True(t, r.IsNack())
		require.Equal(t, receipt.OutcomeDropped, r.Outcome())
		require.Equal(t, "buffer full", r.Reason())
	})
}
