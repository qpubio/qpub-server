package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// Params represents pagination parameters
type Params struct {
	Page    int
	PerPage int
}

// ParseParams parses pagination parameters from query params with defaults
func ParseParams(c *gin.Context) Params {
	// Default values
	defaultPage := 1
	defaultPerPage := 10
	maxPerPage := 100

	// Parse page parameter
	page := defaultPage
	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	// Parse per_page parameter
	perPage := defaultPerPage
	if perPageStr := c.Query("per_page"); perPageStr != "" {
		if parsedPerPage, err := strconv.Atoi(perPageStr); err == nil && parsedPerPage > 0 {
			perPage = parsedPerPage
			// Enforce maximum per_page limit
			if perPage > maxPerPage {
				perPage = maxPerPage
			}
		}
	}

	return Params{
		Page:    page,
		PerPage: perPage,
	}
}
