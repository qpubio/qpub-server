package platform

import (
	"testing"
	"time"

	taskType "github.com/qpubio/qpub-server/internal/shared/type/task"
)

func TestIdempotencyKey(t *testing.T) {
	now := time.Date(2026, 9, 14, 18, 7, 30, 0, time.UTC)
	name := taskType.TaskQueueWorkerCleanupMinutely

	tests := []struct {
		bucket IdempotencyBucket
		want   string
	}{
		{BucketMinute, "queue:worker:cleanup:minutely:202609141807"},
		{BucketTenMinutes, "queue:worker:cleanup:minutely:202609141800"},
		{BucketHour, "queue:worker:cleanup:minutely:2026091418"},
		{BucketDay, "queue:worker:cleanup:minutely:20260914"},
	}

	for _, tt := range tests {
		got := IdempotencyKey(name, tt.bucket, now)
		if got != tt.want {
			t.Errorf("bucket=%d got %q want %q", tt.bucket, got, tt.want)
		}
	}
}
