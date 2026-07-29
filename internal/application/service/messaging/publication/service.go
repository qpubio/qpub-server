package publication

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/domain/messaging/envelope"
	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	"github.com/qpubio/qpub-server/internal/domain/messaging/receipt"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/messaging/router"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

type Service struct {
	logger        logger.Service
	router        domainRouter.Service
	clientService client.Service
}

func NewService(
	logger logger.Service,
	router domainRouter.Service,
	clientService client.Service,
) publication.Service {
	return &Service{
		logger:        logger,
		router:        router,
		clientService: clientService,
	}
}

// Publish publishes a message to a channel via the message router.
func (s *Service) Publish(connID id.ULID, message *publication.Message, skipStats bool) (*publication.PublishResult, error) {
	source := envelope.SourceREST
	if connID != "" {
		source = envelope.SourceWebSocket
	}

	rcpt, result, err := s.router.PublishInbound(context.Background(), domainRouter.PublishRequest{
		ConnectionID:  connID,
		ProjectID:     message.ProjectID,
		Channel:       message.ChannelName,
		Message:       message,
		SkipTelemetry: skipStats,
		Source:        source,
	})
	if err != nil {
		s.sendPublishFailure(connID, message.ChannelName, rcpt)
		return nil, err
	}

	payloads := make([]protocol.DataMessagePayload, len(message.Messages))
	for i, msg := range message.Messages {
		payloads[i] = *protocol.NewDataMessagePayload(msg.Alias, msg.Event, msg.Data)
	}

	s.sendPublishState(
		connID,
		protocol.ActionPublished,
		message.ChannelName,
		payloads,
		0,
		"",
		"",
		0,
		&result.MessageID,
		&result.PublishedAt,
	)

	return result, nil
}

func (s *Service) sendPublishFailure(connID id.ULID, channelName string, rcpt *receipt.Receipt) {
	code := protocol.ErrInternal
	href := protocol.HrefInternal
	message := "Failed to publish message"
	status := protocol.StatusCodeInternal

	if rcpt != nil && rcpt.IsNack() {
		if rcpt.ErrorCode() != 0 {
			code = rcpt.ErrorCode()
		}
		if rcpt.Reason() != "" {
			message = rcpt.Reason()
		}
		switch code {
		case protocol.ErrInvalidMessage:
			href = protocol.HrefInvalidMessage
			status = protocol.StatusCodeBadRequest
		case protocol.ErrPublishFailed:
			href = protocol.HrefPublishFailed
			status = protocol.StatusCodeInternal
		case protocol.ErrRateLimited:
			href = protocol.HrefRateLimited
			status = protocol.StatusCodeTooManyRequests
		}
	}

	s.sendPublishState(
		connID,
		protocol.ActionPublish,
		channelName,
		[]protocol.DataMessagePayload{},
		code,
		href,
		message,
		status,
		nil,
		nil,
	)
}

// sendPublishState sends a state message about the publish operation
func (s *Service) sendPublishState(
	connID id.ULID,
	action protocol.ActionType,
	channelName string,
	messages []protocol.DataMessagePayload,
	errCode protocol.Code,
	errHref protocol.Href,
	errMessage string,
	errStatusCode protocol.StatusCode,
	broadcastMsgID *id.ULID,
	broadcastMsgAt *time.Time,
) error {
	if connID == "" {
		return nil
	}

	var errorInfo *protocol.ErrorInfo
	if errCode != 0 {
		errorInfo = protocol.NewErrorInfo(
			int(errCode),
			string(errHref),
			errMessage,
			int(errStatusCode),
		)
	}

	var stateMsg *protocol.DataMessage
	if errCode == 0 && broadcastMsgID != nil && broadcastMsgAt != nil {
		stateMsg = protocol.NewDataMessageWithEnvelope(
			action,
			channelName,
			messages,
			errorInfo,
			*broadcastMsgID,
			*broadcastMsgAt,
		)
	} else {
		stateMsg = protocol.NewDataMessage(
			action,
			channelName,
			messages,
			errorInfo,
		)
	}

	messageBytes, err := json.Marshal(stateMsg)
	if err != nil {
		s.logger.Error(log.MessagingPublication, "Failed to marshal publish state message channel=%s error=%v",
			channelName, err)
		return fmt.Errorf("failed to marshal publish state message: %w", err)
	}

	if err := s.clientService.SendMessage(connID, messageBytes); err != nil {
		s.logger.Error(log.MessagingPublication, "Failed to send publish state message connectionID=%s channel=%s error=%v",
			connID, channelName, err)
		return fmt.Errorf("failed to send publish state message: %w", err)
	}

	return nil
}
