package job

import (
	"fmt"
	"strings"
	"time"

	"github.com/qpubio/qpub-server/internal/shared/clock"
)

func ValidateCreate(params CreateParams) error {
	if params.ProjectID < 0 {
		return fmt.Errorf("project id is invalid")
	}
	if strings.TrimSpace(params.QueueName) == "" {
		return fmt.Errorf("queue name is required")
	}
	if params.ScheduleAt != nil && params.ScheduleAt.Before(clock.Now().Add(-time.Minute)) {
		return fmt.Errorf("schedule_at is in the past")
	}
	return nil
}
