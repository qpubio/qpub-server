package job

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/qpubio/qpub-server/internal/api/response"
	appDTO "github.com/qpubio/qpub-server/internal/application/dto"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
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
	jobService        domainJob.Service
	queueService      domainQueue.Service
	permissionService permission.Service
}

func NewHandler(
	logger logger.Service,
	router domainRouter.Service,
	jobService domainJob.Service,
	queueService domainQueue.Service,
	permissionService permission.Service,
) *Handler {
	return &Handler{
		logger:            logger,
		router:            router,
		jobService:        jobService,
		queueService:      queueService,
		permissionService: permissionService,
	}
}

type enqueueRequest struct {
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey string          `json:"idempotency_key"`
	ScheduleAt     *time.Time      `json:"schedule_at"`
	Delay          string          `json:"delay"`
	Metadata       json.RawMessage `json:"metadata"`
}

func (h *Handler) Enqueue(c *gin.Context) {
	projectID, permData, ok := h.authContext(c)
	if !ok {
		return
	}

	queueName := c.Param("queueName")
	if !h.canEnqueue(permData, queueName) {
		response.Forbidden(c, "Insufficient permissions")
		return
	}

	var req enqueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	var delay time.Duration
	if d := strings.TrimSpace(req.Delay); d != "" {
		parsed, err := time.ParseDuration(d)
		if err != nil {
			response.BadRequest(c, "Invalid delay duration")
			return
		}
		delay = parsed
	}

	_, jobEntity, err := h.router.Enqueue(c.Request.Context(), domainJob.EnqueueRequest{
		ProjectID:      *projectID,
		QueueName:      queueName,
		Payload:        req.Payload,
		IdempotencyKey: req.IdempotencyKey,
		ScheduleAt:     req.ScheduleAt,
		Delay:          delay,
		Metadata:       req.Metadata,
	})
	if err != nil {
		h.logger.Error(log.Queue, "Enqueue failed: %v", err)
		response.TooManyRequests(c, err.Error())
		return
	}

	response.Created(c, appDTO.NewEnqueueJobResponse(jobEntity))
}

func (h *Handler) Get(c *gin.Context) {
	projectID, permData, ok := h.authContext(c)
	if !ok {
		return
	}
	queueName := c.Param("queueName")
	if !h.canEnqueue(permData, queueName) {
		response.Forbidden(c, "Insufficient permissions")
		return
	}

	jobID := id.ULID(c.Param("jobId"))
	jobEntity, err := h.jobService.Get(*projectID, queueName, jobID)
	if err != nil {
		response.NotFound(c, "Job not found")
		return
	}
	response.OK(c, appDTO.ToJobDTO(jobEntity))
}

func (h *Handler) List(c *gin.Context) {
	projectID, permData, ok := h.authContext(c)
	if !ok {
		return
	}
	queueName := c.Param("queueName")
	if !h.canEnqueue(permData, queueName) {
		response.Forbidden(c, "Insufficient permissions")
		return
	}

	jobs, err := h.jobService.List(domainJob.ListFilter{
		ProjectID: *projectID,
		QueueName: queueName,
		Status:    domainJob.Status(c.Query("status")),
	})
	if err != nil {
		response.InternalError(c, "Failed to list jobs")
		return
	}
	response.OK(c, appDTO.JobsResponse{Jobs: appDTO.ToJobsDTO(jobs)})
}

func (h *Handler) Cancel(c *gin.Context) {
	projectID, permData, ok := h.authContext(c)
	if !ok {
		return
	}
	queueName := c.Param("queueName")
	if !h.canEnqueue(permData, queueName) {
		response.Forbidden(c, "Insufficient permissions")
		return
	}

	jobID := id.ULID(c.Param("jobId"))
	_, jobEntity, err := h.router.Cancel(c.Request.Context(), *projectID, queueName, jobID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, appDTO.ToJobDTO(*jobEntity))
}

func (h *Handler) Retry(c *gin.Context) {
	projectID, permData, ok := h.authContext(c)
	if !ok {
		return
	}
	queueName := c.Param("queueName")
	if !h.canEnqueue(permData, queueName) {
		response.Forbidden(c, "Insufficient permissions")
		return
	}

	jobID := id.ULID(c.Param("jobId"))
	jobEntity, err := h.jobService.Retry(*projectID, queueName, jobID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, appDTO.ToJobDTO(jobEntity))
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

func (h *Handler) canEnqueue(permData []byte, queueName string) bool {
	allowed, err := h.permissionService.CanEnqueue(permData, "queue:"+queueName)
	if err != nil {
		return false
	}
	return allowed
}
