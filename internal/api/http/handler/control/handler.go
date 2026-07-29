package control

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/qpubio/qpub-server/internal/api/response"
	apiKeyDomain "github.com/qpubio/qpub-server/internal/domain/apikey"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
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
	workerService domainWorker.Service
}

func NewHandler(
	tenantService tenant.Service,
	apiKeyService apiKeyDomain.Service,
	queueService domainQueue.Service,
	workerService domainWorker.Service,
) *Handler {
	return &Handler{
		tenantService: tenantService,
		apiKeyService: apiKeyService,
		queueService:  queueService,
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
	c.JSON(http.StatusOK, gin.H{"queues": queues})
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
	c.JSON(http.StatusOK, gin.H{"workers": workers})
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
