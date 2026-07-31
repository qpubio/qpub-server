package control

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/qpubio/qpub-server/internal/api/response"
	"github.com/qpubio/qpub-server/internal/application/dto"
	apiKeyDomain "github.com/qpubio/qpub-server/internal/domain/apikey"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/queue/router"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/gin-gonic/gin"
)

// Handler serves the control API for provisioning tenants, keys, limits, and queue admin.
type Handler struct {
	tenantService tenant.Service
	apiKeyService apiKeyDomain.Service
	queueService  domainQueue.Service
	jobService    domainJob.Service
	router        domainRouter.Service
	workerService domainWorker.Service
}

func NewHandler(
	tenantService tenant.Service,
	apiKeyService apiKeyDomain.Service,
	queueService domainQueue.Service,
	jobService domainJob.Service,
	router domainRouter.Service,
	workerService domainWorker.Service,
) *Handler {
	return &Handler{
		tenantService: tenantService,
		apiKeyService: apiKeyService,
		queueService:  queueService,
		jobService:    jobService,
		router:        router,
		workerService: workerService,
	}
}

type createTenantRequest struct {
	ID id.Int `json:"id" binding:"required"`
}

type setLimitsRequest struct {
	InboundPerSecond  int64 `json:"inbound_per_second"`
	OutboundPerSecond int64 `json:"outbound_per_second"`
}

type createKeyRequest struct {
	Name       string          `json:"name" binding:"required"`
	Permission json.RawMessage `json:"permission"`
	Status     string          `json:"status"`
	ExpiresAt  *time.Time      `json:"expires_at"`
}

type updateKeyRequest struct {
	Name       string          `json:"name" binding:"required"`
	Permission json.RawMessage `json:"permission"` // optional; omitted keeps existing
	Status     string          `json:"status"`
	ExpiresAt  *time.Time      `json:"expires_at"`
}

func (h *Handler) CreateTenant(c *gin.Context) {
	var req createTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	t, err := h.tenantService.Ensure(req.ID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": t.ID, "created_at": t.CreatedAt})
}

func (h *Handler) DeleteTenant(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	// Delete keys scoped to tenant first
	keys, _ := h.apiKeyService.ListByProjectID(tenantID)
	for _, k := range keys {
		_ = h.apiKeyService.Delete(k.ID)
	}
	if err := h.tenantService.Delete(tenantID); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetTenant(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	t, err := h.tenantService.Get(tenantID)
	if err != nil {
		response.NotFound(c, "tenant not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": t.ID, "created_at": t.CreatedAt})
}

func (h *Handler) SetLimits(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	var req setLimitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	l, err := h.tenantService.SetLimits(tenantID, req.InboundPerSecond, req.OutboundPerSecond)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tenant_id":           l.TenantID,
		"inbound_per_second":  l.InboundPerSecond,
		"outbound_per_second": l.OutboundPerSecond,
		"updated_at":          l.UpdatedAt,
	})
}

func (h *Handler) GetLimits(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	l, err := h.tenantService.GetLimits(tenantID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tenant_id":           l.TenantID,
		"inbound_per_second":  l.InboundPerSecond,
		"outbound_per_second": l.OutboundPerSecond,
		"updated_at":          l.UpdatedAt,
	})
}

func (h *Handler) CreateAPIKey(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	if _, err := h.tenantService.Ensure(tenantID); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	var req createKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	status := apiKeyDomain.StatusActive
	if req.Status == string(apiKeyDomain.StatusInactive) {
		status = apiKeyDomain.StatusInactive
	}
	key, err := h.apiKeyService.Create(apiKeyDomain.CreateParams{
		ProjectID:  tenantID,
		Name:       req.Name,
		Permission: req.Permission,
		Status:     status,
		ExpiresAt:  req.ExpiresAt,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         key.ID,
		"public_id":  key.PublicID,
		"project_id": key.ProjectID,
		"name":       key.Name,
		"secret_key": key.SecretKey,
		"permission": apiKeyDomain.EnsurePermission(key.Permission),
		"status":     key.Status,
		"expires_at": key.ExpiresAt,
		"created_at": key.CreatedAt,
	})
}

func (h *Handler) ListAPIKeys(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	keys, err := h.apiKeyService.ListByProjectID(tenantID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	out := make([]gin.H, 0, len(keys))
	for _, k := range keys {
		out = append(out, gin.H{
			"id":         k.ID,
			"public_id":  k.PublicID,
			"project_id": k.ProjectID,
			"name":       k.Name,
			"secret_key": k.SecretKey,
			"permission": apiKeyDomain.EnsurePermission(k.Permission),
			"status":     k.Status,
			"expires_at": k.ExpiresAt,
			"created_at": k.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"keys": out})
}

func (h *Handler) UpdateAPIKey(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	keyID, err := strconv.Atoi(c.Param("keyID"))
	if err != nil {
		response.BadRequest(c, "invalid key id")
		return
	}
	existing, err := h.apiKeyService.Get(id.Int(keyID))
	if err != nil || existing.ProjectID != tenantID {
		response.NotFound(c, "api key not found")
		return
	}
	var req updateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	status := apiKeyDomain.Status(req.Status)
	if status == "" {
		status = existing.Status
	} else if status != apiKeyDomain.StatusActive && status != apiKeyDomain.StatusInactive {
		response.BadRequest(c, "invalid status")
		return
	}
	permission := req.Permission
	if len(permission) == 0 || string(permission) == "null" {
		permission = existing.Permission
	}
	key, err := h.apiKeyService.Update(existing.ID, apiKeyDomain.UpdateParams{
		Name:       req.Name,
		Permission: permission,
		Status:     status,
		ExpiresAt:  req.ExpiresAt,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         key.ID,
		"public_id":  key.PublicID,
		"project_id": key.ProjectID,
		"name":       key.Name,
		"secret_key": key.SecretKey,
		"permission": apiKeyDomain.EnsurePermission(key.Permission),
		"status":     key.Status,
		"expires_at": key.ExpiresAt,
		"created_at": key.CreatedAt,
	})
}

func (h *Handler) DeleteAPIKey(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	keyID, err := strconv.Atoi(c.Param("keyID"))
	if err != nil {
		response.BadRequest(c, "invalid key id")
		return
	}
	key, err := h.apiKeyService.Get(id.Int(keyID))
	if err != nil || key.ProjectID != tenantID {
		response.NotFound(c, "api key not found")
		return
	}
	if err := h.apiKeyService.Delete(key.ID); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListQueues(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	queues, err := h.queueService.List(tenantID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	out := make([]dto.QueueSummaryDTO, 0, len(queues))
	for _, q := range queues {
		counts, err := h.jobService.CountByStatus(tenantID, q.Name)
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		out = append(out, dto.ToQueueSummaryDTO(q, counts))
	}
	response.OK(c, dto.QueuesResponse{Queues: out})
}

func (h *Handler) ListWorkers(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	workers, err := h.workerService.ListByProject(tenantID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, dto.WorkersResponse{Workers: dto.ToWorkersDTO(workers)})
}

func (h *Handler) GetQueue(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	queueName := c.Param("queueName")
	q, err := h.queueService.Get(tenantID, queueName)
	if err != nil {
		if errors.Is(err, domainQueue.ErrNotFound) {
			response.NotFound(c, "queue not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	counts, err := h.jobService.CountByStatus(tenantID, queueName)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, dto.ToQueueSummaryDTO(q, counts))
}

func (h *Handler) ListJobs(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	queueName := c.Param("queueName")
	filter := domainJob.ListFilter{
		ProjectID: tenantID,
		QueueName: queueName,
		Status:    domainJob.Status(c.Query("status")),
	}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			response.BadRequest(c, "invalid limit")
			return
		}
		filter.Limit = n
	}
	if v := c.Query("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			response.BadRequest(c, "invalid offset")
			return
		}
		filter.Offset = n
	}
	jobs, err := h.jobService.List(filter)
	if err != nil {
		response.InternalError(c, "Failed to list jobs")
		return
	}
	response.OK(c, dto.JobsResponse{Jobs: dto.ToJobsDTO(jobs)})
}

func (h *Handler) GetJobCounts(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	queueName := c.Param("queueName")
	counts, err := h.jobService.CountByStatus(tenantID, queueName)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, dto.ToJobCountsDTO(counts))
}

func (h *Handler) GetJob(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	queueName := c.Param("queueName")
	jobID := id.ULID(c.Param("jobId"))
	jobEntity, err := h.jobService.Get(tenantID, queueName, jobID)
	if err != nil {
		if errors.Is(err, domainJob.ErrNotFound) {
			response.NotFound(c, "Job not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, dto.ToJobDTO(jobEntity))
}

func (h *Handler) CancelJob(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	queueName := c.Param("queueName")
	jobID := id.ULID(c.Param("jobId"))
	_, jobEntity, err := h.router.Cancel(c.Request.Context(), tenantID, queueName, jobID)
	if err != nil {
		if errors.Is(err, domainJob.ErrNotFound) {
			response.NotFound(c, "Job not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, dto.ToJobDTO(*jobEntity))
}

func (h *Handler) RetryJob(c *gin.Context) {
	tenantID, err := parseTenantID(c)
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	queueName := c.Param("queueName")
	jobID := id.ULID(c.Param("jobId"))
	jobEntity, err := h.jobService.Retry(tenantID, queueName, jobID)
	if err != nil {
		if errors.Is(err, domainJob.ErrNotFound) {
			response.NotFound(c, "Job not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, dto.ToJobDTO(jobEntity))
}

func (h *Handler) MetricsExport(c *gin.Context) {
	// Placeholder: structured metrics export TBD; telemetry today lives in Redis snapshots.
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "metrics export available via redis telemetry snapshots; structured export TBD",
	})
}

func parseTenantID(c *gin.Context) (id.Int, error) {
	n, err := strconv.Atoi(c.Param("tenantID"))
	if err != nil {
		return 0, err
	}
	return id.Int(n), nil
}
