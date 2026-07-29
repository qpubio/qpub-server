package dto

import (
	"encoding/json"
	"testing"
	"time"

	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

func TestEnqueueJobResponseUsesSnakeCase(t *testing.T) {
	job := domainJob.Job{
		ID:     id.ULID("01JABCDEFGHJKMNPQRSTVWXYZ0"),
		Status: domainJob.StatusPending,
	}

	data, err := json.Marshal(NewEnqueueJobResponse(&job))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := raw["job_id"]; !ok {
		t.Fatalf("expected job_id key, got %s", data)
	}
	if _, ok := raw["jobId"]; ok {
		t.Fatalf("unexpected camelCase jobId in %s", data)
	}
}

func TestJobDTOUsesSnakeCase(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	job := domainJob.Job{
		ID:          id.ULID("01JABCDEFGHJKMNPQRSTVWXYZ0"),
		QueueName:   "emails",
		Status:      domainJob.StatusRunning,
		Attempt:     1,
		MaxAttempts: 25,
		WorkerID:    "worker-1",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(ToJobDTO(job))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	for _, key := range []string{"queue_name", "max_attempts", "worker_id", "created_at", "updated_at"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("expected %q key in %s", key, data)
		}
	}
}
