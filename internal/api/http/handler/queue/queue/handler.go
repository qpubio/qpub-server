package queue

import (
	"github.com/qpubio/qpub-server/internal/api/response"
	appDTO "github.com/qpubio/qpub-server/internal/application/dto"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/permission"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger            logger.Service
	service           domainQueue.Service
	permissionService permission.Service
}

func NewHandler(logger logger.Service, service domainQueue.Service, permissionService permission.Service) *Handler {
	return &Handler{logger: logger, service: service, permissionService: permissionService}
}

type upsertRequest struct {
	ExecutionProfile  string `json:"execution_profile"`
	VisibilityTimeout string `json:"visibility_timeout"`
	MaxAttempts       int    `json:"max_attempts"`
	WebhookURL        string `json:"webhook_url"`
	WebhookSecret     string `json:"webhook_secret"`
}

func (h *Handler) Get(c *gin.Context) {
	projectID, ok := c.Get("projectID")
	if !ok {
		response.Unauthorized(c, "Invalid API key or token")
		return
	}

	queueName := c.Param("queueName")
	q, err := h.service.Get(*projectID.(*id.Int), queueName)
	if err != nil {
		response.NotFound(c, "Queue not found")
		return
	}
	response.OK(c, appDTO.ToQueueDTO(q))
}

func (h *Handler) Upsert(c *gin.Context) {
	projectID, ok := c.Get("projectID")
	if !ok {
		response.Unauthorized(c, "Invalid API key or token")
		return
	}

	queueName := c.Param("queueName")
	var req upsertRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid request body")
			return
		}
	}

	params := domainQueue.CreateParams{
		ProjectID: *projectID.(*id.Int),
		Name:      queueName,
		MaxAttempts: req.MaxAttempts,
		WebhookURL: req.WebhookURL,
		WebhookSecret: req.WebhookSecret,
	}

	q, err := h.service.Ensure(params)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, appDTO.ToQueueDTO(q))
}
