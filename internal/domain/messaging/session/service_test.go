package session_test

import (
	"testing"

	"github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/domain/messaging/session"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/stretchr/testify/require"
)

func TestSessionAggregate(t *testing.T) {
	testsupport.InitRuntime()
	connID := id.NewULID()
	cl := client.New(connID, id.Int(10), id.Int(1), nil, nil)
	sub := subscription.New(cl.ID(), 16)

	s := session.NewSession(connID, id.Int(10), cl, sub)

	require.Equal(t, connID, s.ConnectionID())
	require.Equal(t, id.Int(10), s.ProjectID())
	require.Equal(t, cl, s.Client())
	require.Equal(t, sub, s.Subscription())
	require.Equal(t, sub.ID(), s.SubscriptionID())
	require.True(t, s.IsActive())

	sub.Close()
	require.False(t, s.IsActive())
}
