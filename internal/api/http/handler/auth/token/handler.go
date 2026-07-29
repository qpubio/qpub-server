package token

import (
	"encoding/json"
	"github.com/qpubio/qpub-server/internal/api/response"
	"github.com/qpubio/qpub-server/internal/application/dto"
	"github.com/qpubio/qpub-server/internal/config"
	authToken "github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	tokenService authToken.Service
	logger       logger.Service
	config       *config.Config
}

func NewHandler(tokenService authToken.Service, logger logger.Service, config *config.Config) *Handler {
	return &Handler{
		tokenService: tokenService,
		logger:       logger,
		config:       config,
	}
}

func (h *Handler) IssueToken(c *gin.Context) {
	// Get validated API key details from context (set by APIKeyAuth middleware)
	apiPublicKey := c.MustGet("apiPublicKey").(*string)
	apiSecretKey := c.MustGet("apiSecretKey").(*string)
	defaultPermission := c.MustGet("permission").(*json.RawMessage)

	// Bind optional request body for overrides
	var req dto.IssueTokenRequest
	_ = c.ShouldBindJSON(&req)

	// Apply request overrides or use defaults from context
	permission := req.Permission
	if len(permission) == 0 {
		permission = *defaultPermission
	}

	alias := req.Alias

	// Determine expiration duration
	expiresIn := int(h.config.Auth.Token.APIKey.Duration.Seconds())
	if req.ExpiresIn != nil {
		expiresIn = *req.ExpiresIn
		if expiresIn <= 0 {
			response.BadRequest(c, "expiresIn must be greater than 0")
			return
		}
		maxDuration := int(h.config.Auth.Token.APIKey.MaxDuration.Seconds())
		if expiresIn > maxDuration {
			response.BadRequest(c, "expiresIn exceeds maximum allowed duration")
			return
		}
	}

	// Generate token
	token, err := h.tokenService.CreateAPIKeyToken(
		*apiPublicKey,
		permission,
		alias,
		[]byte(*apiSecretKey),
		time.Duration(expiresIn)*time.Second,
	)
	if err != nil {
		h.logger.Error("auth-token-handler", "Failed to create API key token: %v", err)
		response.InternalError(c, "Failed to generate token")
		return
	}

	response.OK(c, dto.TokenResponse{Token: token})
}

func (h *Handler) RequestToken(c *gin.Context) {
	apiPublicKey := c.MustGet("apiPublicKey").(*string)
	apiSecretKey := c.MustGet("apiSecretKey").(*string)
	permission := c.MustGet("permission").(*json.RawMessage)
	alias := c.MustGet("alias").(*string)

	expiresIn := h.config.Auth.Token.APIKey.Duration
	token, err := h.tokenService.CreateAPIKeyToken(
		*apiPublicKey,
		*permission,
		*alias,
		[]byte(*apiSecretKey),
		expiresIn,
	)
	if err != nil {
		h.logger.Error("auth-token-handler", "Failed to create API key token from token request: %v", err)
		response.InternalError(c, "Failed to generate token")
		return
	}

	response.OK(c, dto.TokenResponse{Token: token})
}
