package router_test

import (
	"context"
	"testing"

	backpressureApp "github.com/qpubio/qpub-server/internal/application/service/messaging/backpressure"
	routerApp "github.com/qpubio/qpub-server/internal/application/service/messaging/router"
	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/messaging/envelope"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/messaging/router"
	domainPublication "github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/stretchr/testify/require"
)

type rateLimitLogger struct{}

func (rateLimitLogger) Debug(log.Component, string, ...interface{}) {}
func (rateLimitLogger) Info(log.Component, string, ...interface{})  {}
func (rateLimitLogger) Warn(log.Component, string, ...interface{})  {}
func (rateLimitLogger) Error(log.Component, string, ...interface{}) {}
func (rateLimitLogger) Fatal(log.Component, string, ...interface{}) {}

type fixedLimits struct {
	limits backpressure.MessageRateLimits
}

func (f fixedLimits) MessageRates(id.Int) (backpressure.MessageRateLimits, error) {
	return f.limits, nil
}

func TestPublishInbound_RejectsWhenInboundRateLimited(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	gatekeeper := backpressureApp.NewGatekeeperService(rateLimitLogger{}, fixedLimits{
		limits: backpressure.MessageRateLimits{InboundPerSecond: 1},
	})

	router := routerApp.NewService(
		rateLimitLogger{},
		instanceID,
		newStubBroker(),
		&stubChannelSvc{instanceID: instanceID},
		newMemorySubscriptionRepo(),
		newCapturingDeliverer(),
		stubTelemetry{},
		gatekeeper,
	)

	msg, err := domainPublication.Create(domainPublication.CreateParams{
		ProjectID:   id.Int(7),
		ChannelName: "room",
		Messages: []domainPublication.Payload{{
			Event: strPtr("evt"),
			Data:  map[string]string{"k": "v"},
		}},
	})
	require.NoError(t, err)

	_, _, err = router.PublishInbound(context.Background(), domainRouter.PublishRequest{
		ProjectID: id.Int(7),
		Channel:   "room",
		Message:   msg,
		Source:    envelope.SourceREST,
	})
	require.NoError(t, err)

	_, _, err = router.PublishInbound(context.Background(), domainRouter.PublishRequest{
		ProjectID: id.Int(7),
		Channel:   "room",
		Message:   msg,
		Source:    envelope.SourceREST,
	})
	require.ErrorIs(t, err, domainPublication.ErrRateLimited)
}

func strPtr(v string) *string { return &v }
