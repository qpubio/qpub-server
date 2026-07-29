package job

import (
	"os"
	"testing"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

func TestMain(m *testing.M) {
	testsupport.InitRuntime()
	os.Exit(m.Run())
}

func TestJobLifecycle(t *testing.T) {
	j, err := Enqueue(CreateParams{
		ProjectID: id.Int(1),
		QueueName: "reports",
		Payload:   []byte(`{"task":"export"}`),
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if j.Status != StatusPending {
		t.Fatalf("expected pending, got %s", j.Status)
	}

	j.MarkRunning("worker-1")
	if j.Status != StatusRunning || j.Attempt != 1 {
		t.Fatalf("unexpected running state: %+v", j)
	}

	j.MarkCompleted([]byte(`{"ok":true}`))
	if j.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", j.Status)
	}
}

func TestJobScheduled(t *testing.T) {
	future := clock.Now().Add(time.Hour)
	j, err := Enqueue(CreateParams{
		ProjectID:  id.Int(1),
		QueueName:  "delayed",
		ScheduleAt: &future,
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if j.Status != StatusScheduled {
		t.Fatalf("expected scheduled, got %s", j.Status)
	}
	if j.IsClaimable(clock.Now()) {
		t.Fatal("scheduled job should not be claimable yet")
	}
}

func TestJobRetryAndDLQ(t *testing.T) {
	j, _ := Enqueue(CreateParams{
		ProjectID:   id.Int(1),
		QueueName:   "retry-test",
		MaxAttempts: 2,
	})
	j.MarkRunning("w1")
	j.MarkRetry(time.Minute)
	if !j.CanRetry() {
		t.Fatal("should retry")
	}

	j.MarkRunning("w1")
	j.MarkDLQ("permanent failure")
	if j.Status != StatusDLQ {
		t.Fatalf("expected dlq, got %s", j.Status)
	}
}

func TestJobDueScheduledIsClaimable(t *testing.T) {
	past := clock.Now().Add(-30 * time.Second)
	j, err := Enqueue(CreateParams{
		ProjectID:  id.Int(1),
		QueueName:  "delayed",
		ScheduleAt: &past,
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	// Past schedule_at still creates scheduled if After(now) was true at create —
	// force scheduled + past for the claimability check.
	j.Status = StatusScheduled
	j.ScheduleAt = &past
	if !j.IsClaimable(clock.Now()) {
		t.Fatal("due scheduled job should be claimable")
	}
	j.MarkPendingForSchedule()
	if j.Status != StatusPending {
		t.Fatalf("expected pending after promote, got %s", j.Status)
	}
}

func TestJobMarkReclaimed(t *testing.T) {
	j, _ := Enqueue(CreateParams{
		ProjectID: id.Int(1),
		QueueName: "reclaim-test",
	})
	j.MarkRunning("w1")
	attempt := j.Attempt
	j.MarkReclaimed()
	if j.Status != StatusPending {
		t.Fatalf("expected pending, got %s", j.Status)
	}
	if j.WorkerID != "" || j.StartedAt != nil {
		t.Fatalf("expected cleared lease fields, got worker=%q started=%v", j.WorkerID, j.StartedAt)
	}
	if j.Attempt != attempt {
		t.Fatalf("attempt should be preserved, got %d want %d", j.Attempt, attempt)
	}
	if !j.IsClaimable(clock.Now()) {
		t.Fatal("reclaimed job should be claimable")
	}
}
