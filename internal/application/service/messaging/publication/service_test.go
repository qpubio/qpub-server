package publication

import (
	"context"
	"encoding/json"
	"testing"

	routerApp "github.com/qpubio/qpub-server/internal/application/service/messaging/router"
	sharedcfg "github.com/qpubio/qpub-server/internal/config/shared"
	"github.com/qpubio/qpub-server/internal/domain/messaging/broker"
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	domainPublication "github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/shared/clock"
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

type stubBroker struct {
	lastSubject string
	lastPayload []byte
	handlers    map[string]broker.MessageHandler
}

func newStubBroker() *stubBroker {
	return &stubBroker{handlers: make(map[string]broker.MessageHandler)}
}

func (s *stubBroker) ListenToChannel(channelName string, handler broker.MessageHandler) error {
	s.handlers[channelName] = handler
	return nil
}
func (s *stubBroker) StopListeningToChannel(string) error { return nil }
func (s *stubBroker) PublishToChannel(subject string, message []byte) error {
	s.lastSubject = subject
	s.lastPayload = append([]byte(nil), message...)
	if handler, ok := s.handlers[subject]; ok {
		handler(subject, append([]byte(nil), message...))
	}
	return nil
}
func (s *stubBroker) Shutdown(context.Context) error { return nil }

type stubChannelSvc struct {
	instanceID id.ULID
}

func (s *stubChannelSvc) Create(string, id.Int) (*channel.Channel, error) { panic("unexpected") }
func (s *stubChannelSvc) Update(*channel.Channel) error                    { return nil }
func (s *stubChannelSvc) Delete(string, id.Int) error                     { panic("unexpected") }
func (s *stubChannelSvc) Get(rawName string, projID id.Int) (*channel.Channel, error) {
	return channel.New(channel.NewName(rawName, projID), s.instanceID), nil
}
func (s *stubChannelSvc) GetOrCreate(rawName string, projID id.Int) (*channel.Channel, error) {
	return channel.New(channel.NewName(rawName, projID), s.instanceID), nil
}

type emptySubscriptionRepo struct{}

func (emptySubscriptionRepo) AddToChannel(string, *subscription.Subscription) error { return nil }
func (emptySubscriptionRepo) RemoveFromChannel(string, *subscription.Subscription) error {
	return nil
}
func (emptySubscriptionRepo) RemoveSubscriptionFromAll(*subscription.Subscription) error { return nil }
func (emptySubscriptionRepo) GetAllLocalForChannel(string) ([]*subscription.Subscription, error) {
	return nil, nil
}
func (emptySubscriptionRepo) GetAllChannelsForSubscription(*subscription.Subscription) ([]string, error) {
	return nil, nil
}
func (emptySubscriptionRepo) CleanClosedSubscriptions() int { return 0 }
func (emptySubscriptionRepo) FindByID(id.ULID) (*subscription.Subscription, error) {
	return nil, subscription.ErrNil
}

type noopDeliverer struct{}

func (noopDeliverer) Deliver(id.ULID, []byte) error { return nil }

type stubTelemetry struct{}

func (stubTelemetry) Record(*domainTelemetry.Event) error { return nil }
func (stubTelemetry) GetSnapshots(id.ULID) ([]*domainTelemetry.Snapshot, error) {
	return nil, nil
}
func (stubTelemetry) ResetForProject(id.Int) error { return nil }

type capturingClient struct {
	last [][]byte
}

func (capturingClient) Connect(id.ULID, id.Int, id.Int, *string, *json.RawMessage) (*client.Client, error) {
	panic("unexpected")
}
func (capturingClient) Disconnect(id.ULID) error                  { panic("unexpected") }
func (capturingClient) GetClient(id.ULID) (*client.Client, error) { panic("unexpected") }
func (c *capturingClient) SendMessage(_ id.ULID, msg []byte) error {
	c.last = append(c.last, append([]byte(nil), msg...))
	return nil
}
func (capturingClient) BroadcastToProject(id.Int, []byte) error { panic("unexpected") }
func (capturingClient) CleanDisconnectedClients() (int, error)  { panic("unexpected") }

func initTestDeps(t *testing.T) {
	t.Helper()
	clock.Init()
	cfg := &sharedcfg.ID{
		HashSalt:   "test-salt-for-testing-12345",
		HashLength: 8,
		ULIDLength: 22,
	}
	if err := id.Init(cfg); err != nil {
		t.Logf("id.Init: %v (may already be initialized)", err)
	}
}

type allowAllGatekeeper struct{}

func (allowAllGatekeeper) AllowInbound(id.Int) bool  { return true }
func (allowAllGatekeeper) AllowOutbound(id.Int) bool { return true }

func newTestPublicationService(inst id.ULID, br *stubBroker, cl client.Service) domainPublication.Service {
	router := routerApp.NewService(
		nopLogger{},
		inst,
		br,
		&stubChannelSvc{instanceID: inst},
		emptySubscriptionRepo{},
		noopDeliverer{},
		stubTelemetry{},
		allowAllGatekeeper{},
	)
	return NewService(nopLogger{}, router, cl)
}

func TestPublish_ReturnsBroadcastEnvelopeMetadata(t *testing.T) {
	initTestDeps(t)

	br := newStubBroker()
	inst := id.NewULID()
	svc := newTestPublicationService(inst, br, &capturingClient{})

	ev := "evt"
	msg, err := domainPublication.Create(domainPublication.CreateParams{
		ProjectID:   7,
		ChannelName: "room-a",
		Messages: []domainPublication.Payload{{
			Event: &ev,
			Data:  map[string]string{"k": "v"},
		}},
	})
	require.NoError(t, err)

	res, err := svc.Publish("", msg, false)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "room-a", res.Channel)
	require.Equal(t, 1, res.PayloadCount)
	require.NotEmpty(t, res.MessageID)
	require.False(t, res.PublishedAt.IsZero())

	var wire protocol.DataMessage
	require.NoError(t, json.Unmarshal(br.lastPayload, &wire))
	require.Equal(t, protocol.ActionMessage, wire.Action)
	require.Equal(t, res.MessageID, wire.ID)
	require.Equal(t, res.PublishedAt.UTC(), wire.Timestamp.UTC())
	require.Equal(t, "room-a", wire.Channel)
	require.Len(t, wire.Messages, 1)
	require.Equal(t, "7.room-a", br.lastSubject)
}

func TestPublish_WebSocketAckMatchesBroadcastEnvelope(t *testing.T) {
	initTestDeps(t)

	br := newStubBroker()
	cl := &capturingClient{}
	inst := id.NewULID()
	svc := newTestPublicationService(inst, br, cl)

	ev := "evt"
	msg, err := domainPublication.Create(domainPublication.CreateParams{
		ProjectID:   7,
		ChannelName: "room-a",
		Messages: []domainPublication.Payload{{
			Event: &ev,
			Data:  map[string]string{"k": "v"},
		}},
	})
	require.NoError(t, err)

	connID := id.NewULID()
	_, err = svc.Publish(connID, msg, false)
	require.NoError(t, err)
	require.Len(t, cl.last, 1)

	var broadcast protocol.DataMessage
	require.NoError(t, json.Unmarshal(br.lastPayload, &broadcast))
	require.Equal(t, protocol.ActionMessage, broadcast.Action)

	var ack protocol.DataMessage
	require.NoError(t, json.Unmarshal(cl.last[0], &ack))
	require.Equal(t, protocol.ActionPublished, ack.Action)
	require.Equal(t, broadcast.ID, ack.ID)
	require.Equal(t, broadcast.Timestamp.UTC(), ack.Timestamp.UTC())
	require.Equal(t, broadcast.Channel, ack.Channel)
	require.Len(t, ack.Messages, len(broadcast.Messages))
}
