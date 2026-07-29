package queue

import (
	"fmt"
	"strings"
)

func ValidateCreate(params CreateParams) error {
	if params.ProjectID < 0 {
		return fmt.Errorf("project id is invalid")
	}
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return fmt.Errorf("queue name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("queue name too long")
	}
	return nil
}
