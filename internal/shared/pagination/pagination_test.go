package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestParseParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		queryParams string
		expected    Params
	}{
		{
			name:        "no query params - should use defaults",
			queryParams: "",
			expected:    Params{Page: 1, PerPage: 10},
		},
		{
			name:        "valid page and per_page",
			queryParams: "?page=2&per_page=20",
			expected:    Params{Page: 2, PerPage: 20},
		},
		{
			name:        "only page provided",
			queryParams: "?page=3",
			expected:    Params{Page: 3, PerPage: 10},
		},
		{
			name:        "only per_page provided",
			queryParams: "?per_page=25",
			expected:    Params{Page: 1, PerPage: 25},
		},
		{
			name:        "invalid page - should use default",
			queryParams: "?page=invalid&per_page=15",
			expected:    Params{Page: 1, PerPage: 15},
		},
		{
			name:        "invalid per_page - should use default",
			queryParams: "?page=2&per_page=invalid",
			expected:    Params{Page: 2, PerPage: 10},
		},
		{
			name:        "zero page - should use default",
			queryParams: "?page=0&per_page=15",
			expected:    Params{Page: 1, PerPage: 15},
		},
		{
			name:        "negative page - should use default",
			queryParams: "?page=-1&per_page=15",
			expected:    Params{Page: 1, PerPage: 15},
		},
		{
			name:        "zero per_page - should use default",
			queryParams: "?page=2&per_page=0",
			expected:    Params{Page: 2, PerPage: 10},
		},
		{
			name:        "negative per_page - should use default",
			queryParams: "?page=2&per_page=-5",
			expected:    Params{Page: 2, PerPage: 10},
		},
		{
			name:        "per_page exceeds maximum - should cap at 100",
			queryParams: "?page=1&per_page=150",
			expected:    Params{Page: 1, PerPage: 100},
		},
		{
			name:        "large valid values",
			queryParams: "?page=100&per_page=50",
			expected:    Params{Page: 100, PerPage: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test HTTP request
			req := httptest.NewRequest(http.MethodGet, "/test"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Create gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Parse pagination parameters
			result := ParseParams(c)

			// Assert the result
			assert.Equal(t, tt.expected, result)
		})
	}
}
