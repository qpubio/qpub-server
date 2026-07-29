package worker

import (
	"github.com/qpubio/qpub-server/internal/api/response"
	appDTO "github.com/qpubio/qpub-server/internal/application/dto"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger  logger.Service
	service domainWorker.Service
}

func NewHandler(logger logger.Service, service domainWorker.Service) *Handler {
	return &Handler{logger: logger, service: service}
}

type registerRequest struct {
	Name   string   `json:"name" binding:"required"`
	Queues []string `json:"queues"`
}

type heartbeatRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	projectID, ok := c.Get("projectID")
	if !ok {
		response.Unauthorized(c, "Invalid API key or token")
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	w, err := h.service.Register(domainWorker.CreateParams{
		ProjectID: *projectID.(*id.Int),
		Name:      req.Name,
		Queues:    req.Queues,
	})
	if err != nil {
		response.InternalError(c, "Failed to register worker")
		return
	}

	response.Created(c, appDTO.ToWorkerDTO(w))
}

func (h *Handler) Heartbeat(c *gin.Context) {
	projectID, ok := c.Get("projectID")
	if !ok {
		response.Unauthorized(c, "Invalid API key or token")
		return
	}

	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	w, err := h.service.Heartbeat(*projectID.(*id.Int), id.ULID(req.WorkerID))
	if err != nil {
		response.NotFound(c, "Worker not found")
		return
	}

	response.OK(c, appDTO.ToWorkerDTO(w))
}
