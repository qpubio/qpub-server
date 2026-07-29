package pull

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/qpubio/qpub-server/internal/api/response"
	appDTO "github.com/qpubio/qpub-server/internal/application/dto"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/queue/router"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/permission"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger            logger.Service
	router            domainRouter.Service
	permissionService permission.Service
}

func NewHandler(logger logger.Service, router domainRouter.Service, permissionService permission.Service) *Handler {
	return &Handler{logger: logger, router: router, permissionService: permissionService}
}

type pullRequest struct {
	WorkerID  string `json:"worker_id" binding:"required"`
	BatchSize int    `json:"batch_size"`
	Wait      string `json:"wait"`
}

type ackRequest struct {
	WorkerID string          `json:"worker_id" binding:"required"`
	Result   json.RawMessage `json:"result"`
}

type nackRequest struct {
	WorkerID   string `json:"worker_id" binding:"required"`
	Reason     string `json:"reason"`
	RetryDelay string `json:"retry_delay"`
}

func (h *Handler) Pull(c *gin.Context) {
	projectID, permData, ok := h.authContext(c)
	if !ok {
		return
	}
	queueName := c.Param("queueName")
	if !h.canDequeue(permData, queueName) {
		response.Forbidden(c, "Insufficient permissions")
		return
	}

	var req pullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	wait := 20 * time.Second
	if req.Wait != "" {
		parsed, err := time.ParseDuration(req.Wait)
		if err == nil {
			wait = parsed
		}
	}

	jobs, err := h.router.Dequeue(c.Request.Context(), domainJob.DequeueRequest{
		ProjectID: *projectID,
		QueueName: queueName,
		WorkerID:  req.WorkerID,
		BatchSize: req.BatchSize,
		Wait:      wait,
	})
	if err != nil {
		h.logger.Error(log.Queue, "Pull failed: %v", err)
		response.InternalError(c, "Failed to pull jobs")
		return
	}

	response.OK(c, appDTO.JobsResponse{Jobs: appDTO.ToJobsDTO(jobs)})
}

func (h *Handler) Ack(c *gin.Context) {
	projectID, permData, ok := h.authContext(c)
	if !ok {
		return
	}
	queueName := c.Param("queueName")
	if !h.canDequeue(permData, queueName) {
		response.Forbidden(c, "Insufficient permissions")
		return
	}

	var req ackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	jobID := id.ULID(c.Param("jobId"))
	_, err := h.router.Ack(c.Request.Context(), domainJob.AckRequest{
		ProjectID: *projectID,
		QueueName: queueName,
		JobID:     jobID,
		WorkerID:  req.WorkerID,
		Result:    req.Result,
	})
	if err != nil {
		h.writeJobActionError(c, err)
		return
	}
	response.OK(c, gin.H{"status": "acknowledged"})
}

func (h *Handler) Nack(c *gin.Context) {
	projectID, permData, ok := h.authContext(c)
	if !ok {
		return
	}
	queueName := c.Param("queueName")
	if !h.canDequeue(permData, queueName) {
		response.Forbidden(c, "Insufficient permissions")
		return
	}

	var req nackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	var retryDelay time.Duration
	if req.RetryDelay != "" {
		parsed, err := time.ParseDuration(req.RetryDelay)
		if err == nil {
			retryDelay = parsed
		}
	}

	jobID := id.ULID(c.Param("jobId"))
	_, err := h.router.Nack(c.Request.Context(), domainJob.NackRequest{
		ProjectID:  *projectID,
		QueueName:  queueName,
		JobID:      jobID,
		WorkerID:   req.WorkerID,
		Reason:     req.Reason,
		RetryDelay: retryDelay,
	})
	if err != nil {
		h.writeJobActionError(c, err)
		return
	}
	response.OK(c, gin.H{"status": "nacked"})
}

func (h *Handler) writeJobActionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainJob.ErrNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, domainJob.ErrWorkerMismatch):
		response.Forbidden(c, err.Error())
	case errors.Is(err, domainJob.ErrInvalidTransition):
		response.Conflict(c, err.Error())
	default:
		response.BadRequest(c, err.Error())
	}
}

func (h *Handler) authContext(c *gin.Context) (*id.Int, []byte, bool) {
	ptr, ok := c.Get("projectID")
	if !ok {
		response.Unauthorized(c, "Invalid API key or token")
		return nil, nil, false
	}
	projectID := ptr.(*id.Int)

	permPtr, ok := c.Get("permission")
	if !ok {
		response.Unauthorized(c, "Invalid API key or token")
		return nil, nil, false
	}
	permRaw, ok := permPtr.(*json.RawMessage)
	if !ok || permRaw == nil {
		response.Unauthorized(c, "Invalid API key or token")
		return nil, nil, false
	}
	return projectID, []byte(*permRaw), true
}

func (h *Handler) canDequeue(permData []byte, queueName string) bool {
	allowed, err := h.permissionService.CanDequeue(permData, "queue:"+queueName)
	if err != nil {
		return false
	}
	return allowed
}
