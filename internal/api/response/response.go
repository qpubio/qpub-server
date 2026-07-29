package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error represents error information
type Error struct {
	Code    string      `json:"code"`              // Machine-readable error code
	Message string      `json:"message"`           // Human-readable error message
	Details interface{} `json:"details,omitempty"` // Optional error details
}

// Standard response functions

func OK(c *gin.Context, data interface{}) {
	send(c, http.StatusOK, data)
}

func Created(c *gin.Context, data interface{}) {
	send(c, http.StatusCreated, data)
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error response functions

func BadRequest(c *gin.Context, message string, details ...interface{}) {
	sendError(c, http.StatusBadRequest, "BAD_REQUEST", message, details...)
}

func Unauthorized(c *gin.Context, message string, details ...interface{}) {
	sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", message, details...)
}

func Forbidden(c *gin.Context, message string, details ...interface{}) {
	sendError(c, http.StatusForbidden, "FORBIDDEN", message, details...)
}

func NotFound(c *gin.Context, message string, details ...interface{}) {
	sendError(c, http.StatusNotFound, "NOT_FOUND", message, details...)
}

func Conflict(c *gin.Context, message string, details ...interface{}) {
	sendError(c, http.StatusConflict, "CONFLICT", message, details...)
}

func InternalError(c *gin.Context, message string, details ...interface{}) {
	sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, details...)
}

func PartialContent(c *gin.Context, message string, details ...interface{}) {
	sendError(c, http.StatusPartialContent, "PARTIAL_CONTENT", message, details...)
}

func TooManyRequests(c *gin.Context, message string, details ...interface{}) {
	sendError(c, http.StatusTooManyRequests, "RATE_LIMITED", message, details...)
}

// Helper functions

func send(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

func sendError(c *gin.Context, status int, code, message string, details ...interface{}) {
	var detailsData interface{}
	if len(details) > 0 {
		detailsData = details[0]
	}

	send(c, status, &Error{
		Code:    code,
		Message: message,
		Details: detailsData,
	})
}
