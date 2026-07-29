# Pagination Package

This package provides utilities for handling pagination in HTTP handlers.

## ParseParams

The `ParseParams` function extracts pagination parameters from query parameters with sensible defaults and validation.

### Usage

```go
import "qpub/internal/shared/pagination"

func (h *Handler) List(c *gin.Context) {
    // Parse pagination parameters from query params
    params := pagination.ParseParams(c)
    
    // Use the parsed parameters
    results, total, err := h.service.List(params.Page, params.PerPage)
    // ... handle results
}
```

### Query Parameters

- `page` - The current page number (1-based)
- `per_page` - Number of items per page

### Defaults and Validation

- **Default page**: 1
- **Default per_page**: 10
- **Maximum per_page**: 100 (values above this are capped)
- **Minimum values**: Both page and per_page must be positive integers
- **Invalid values**: Fall back to defaults

### Examples

| Query String | Result |
|--------------|--------|
| `?page=2&per_page=20` | `{Page: 2, PerPage: 20}` |
| `?page=3` | `{Page: 3, PerPage: 10}` |
| `?per_page=25` | `{Page: 1, PerPage: 25}` |
| `?page=0&per_page=150` | `{Page: 1, PerPage: 100}` |
| `?page=invalid` | `{Page: 1, PerPage: 10}` |
| (no params) | `{Page: 1, PerPage: 10}` |

### Standard Parameter Names

This utility follows common REST API conventions:
- `page` for the current page number
- `per_page` for the number of items per page

These names are consistent with the pagination response format used in the application DTOs. 