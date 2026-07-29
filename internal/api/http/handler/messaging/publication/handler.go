package publication

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/qpubio/qpub-server/internal/api/response"
	"github.com/qpubio/qpub-server/internal/application/dto"
	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	domainPublication "github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/permission"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger            logger.Service
	service           domainPublication.Service
	permissionService permission.Service
}

func NewHandler(
	logger logger.Service,
	service domainPublication.Service,
	permissionService permission.Service,
) *Handler {
	return &Handler{
		service:           service,
		logger:            logger,
		permissionService: permissionService,
	}
}

func (h *Handler) Publish(c *gin.Context) {
	ptr, ok := c.Get("projectID")
	if !ok {
		response.Unauthorized(c, "Invalid API key or token")
		return
	}
	projectID := ptr.(*id.Int)

	var params protocol.RESTPublishMessage
	if err := c.ShouldBindJSON(&params); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	// Validate messages array is not empty
	if len(params.Messages) == 0 {
		response.BadRequest(c, "At least one message is required")
		return
	}

	var channels []*string
	if channelName := c.Param("channelName"); channelName != "" {
		channels = []*string{&channelName}
	} else {
		channels = params.Channels
	}

	if len(channels) == 0 {
		response.BadRequest(c, "Channel name is required")
		return
	}

	permPtr, ok := c.Get("permission")
	if !ok {
		response.Unauthorized(c, "Invalid API key or token")
		return
	}
	permRaw, ok := permPtr.(*json.RawMessage)
	if !ok || permRaw == nil {
		response.Unauthorized(c, "Invalid API key or token")
		return
	}
	permData := []byte(*permRaw)
	for _, channel := range channels {
		canPublish, err := h.permissionService.CanPublish(permData, *channel)
		if err != nil {
			h.logger.Error(log.PublicationHandler, `Error checking publish permissions channel=%v error=%v`, channel,
				err)
			response.BadRequest(c, "Invalid permission format")
			return
		}
		if !canPublish {
			h.logger.Warn(log.PublicationHandler, `Publish permission denied channel=%v`, channel)
			response.Forbidden(c, fmt.Sprintf("Permission denied to publish to channel: %s", *channel))
			return
		}
	}

	// Transform all messages from request to domain payloads once (performance optimization)
	var payloads []domainPublication.Payload
	for _, msg := range params.Messages {
		payloads = append(payloads, domainPublication.Payload{
			Alias: msg.Alias,
			Event: msg.Event,
			Data:  msg.Data,
		})
	}

	var published []domainPublication.PublishResult
	var failedChannels []string
	for _, channel := range channels {
		message, err := domainPublication.Create(domainPublication.CreateParams{
			ProjectID:   *projectID,
			ChannelName: *channel,
			Messages:    payloads,
		})
		if err != nil {
			failedChannels = append(failedChannels, *channel)
			h.logger.Error(log.PublicationHandler, `Failed to create publication message channel=%v error=%v`, channel,
				err)
			continue
		}

		res, err := h.service.Publish("", message, false)
		if err != nil {
			if errors.Is(err, domainPublication.ErrRateLimited) {
				response.TooManyRequests(c, "Rate limit exceeded")
				return
			}
			failedChannels = append(failedChannels, *channel)
			h.logger.Error(log.PublicationHandler, `Failed to publish message channel=%v error=%v`, channel,
				err)
			continue
		}
		published = append(published, *res)
	}

	if len(failedChannels) > 0 {
		if len(failedChannels) == len(channels) {
			response.InternalError(c, "Failed to publish to all channels")
			return
		}
		response.PartialContent(c, "Failed to publish to some channels",
			map[string]interface{}{
				"published":       dto.NewPublishMessageResponse(published).Published,
				"failed_channels": failedChannels,
			})
		return
	}

	response.Created(c, dto.NewPublishMessageResponse(published))
}
