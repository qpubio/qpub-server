package backpressure_test

import (
	"testing"

	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/messaging/envelope"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/stretchr/testify/require"
)

func TestBoundedQueue(t *testing.T) {
	q := backpressure.NewBoundedQueue(2)
	require.True(t, q.Push([]byte("a")))
	require.True(t, q.Push([]byte("b")))
	require.False(t, q.Push([]byte("c")))
	require.Equal(t, 2, q.Depth())

	item, ok := q.Pop()
	require.True(t, ok)
	require.Equal(t, []byte("a"), item)
	require.Equal(t, 1, q.Depth())
}

func TestDropNewestPolicy(t *testing.T) {
	testsupport.InitRuntime()
	q := backpressure.NewBoundedQueue(1)
	policy := backpressure.NewDropNewestPolicy(backpressure.PointSubscriptionBuffer)

	env := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionOutbound,
		ProjectID: id.Int(1),
		Channel:   "room",
		Payload:   []byte("first"),
		Source:    envelope.SourceNATS,
	})

	result := policy.TryEnqueue(q, env)
	require.True(t, result.Queued)
	require.False(t, result.Dropped)

	env2 := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionOutbound,
		ProjectID: id.Int(1),
		Channel:   "room",
		Payload:   []byte("second"),
		Source:    envelope.SourceNATS,
	})
	result = policy.TryEnqueue(q, env2)
	require.True(t, result.Dropped)
	require.Equal(t, backpressure.DropReasonBufferFull, result.Reason)
	require.Equal(t, 1, q.Depth())
	require.Equal(t, []byte("first"), q.Items()[0])
}

func TestBlockWithTimeoutPolicyFallback(t *testing.T) {
	testsupport.InitRuntime()
	q := backpressure.NewBoundedQueue(1)
	policy := backpressure.NewBlockWithTimeoutPolicy(backpressure.PointConnectionOutbound, 50)

	env := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionOutbound,
		ProjectID: id.Int(1),
		Channel:   "room",
		Payload:   []byte("first"),
		Source:    envelope.SourceNATS,
	})
	require.True(t, policy.TryEnqueue(q, env).Queued)

	env2 := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionOutbound,
		ProjectID: id.Int(1),
		Channel:   "room",
		Payload:   []byte("second"),
		Source:    envelope.SourceNATS,
	})
	result := policy.TryEnqueue(q, env2)
	require.True(t, result.Dropped)
	require.Equal(t, backpressure.DropReasonWriteTimeout, result.Reason)
}
