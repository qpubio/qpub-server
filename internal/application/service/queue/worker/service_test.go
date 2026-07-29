package worker

import (
	"os"
	"testing"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	logType "github.com/qpubio/qpub-server/internal/shared/type/log"
)

func TestMain(m *testing.M) {
	testsupport.InitRuntime()
	os.Exit(m.Run())
}

type mockWorkerRepo struct {
	byID map[id.ULID]*domainWorker.Worker
}

func (m *mockWorkerRepo) Create(w *domainWorker.Worker) error {
	if m.byID == nil {
		m.byID = make(map[id.ULID]*domainWorker.Worker)
	}
	cp := *w
	m.byID[w.ID] = &cp
	return nil
}

func (m *mockWorkerRepo) Update(w *domainWorker.Worker) error {
	cp := *w
	m.byID[w.ID] = &cp
	return nil
}

func (m *mockWorkerRepo) FindByID(_ id.Int, workerID id.ULID) (*domainWorker.Worker, error) {
	w, ok := m.byID[workerID]
	if !ok {
		return nil, domainWorker.ErrNotFound
	}
	cp := *w
	return &cp, nil
}

func (m *mockWorkerRepo) ListByProject(id.Int) ([]domainWorker.Worker, error) {
	return nil, nil
}

type mockJobRepo struct {
	running []domainJob.Job
}

func (m *mockJobRepo) Create(*domainJob.Job) error { return nil }
func (m *mockJobRepo) Update(*domainJob.Job) error { return nil }
func (m *mockJobRepo) FindByID(id.Int, string, id.ULID) (*domainJob.Job, error) {
	return nil, domainJob.ErrNotFound
}
func (m *mockJobRepo) FindByIdempotencyKey(id.Int, string, string) (*domainJob.Job, error) {
	return nil, domainJob.ErrNotFound
}
func (m *mockJobRepo) List(domainJob.ListFilter) ([]domainJob.Job, error) { return nil, nil }
func (m *mockJobRepo) FindDueScheduled(int, time.Time) ([]domainJob.Job, error) {
	return nil, nil
}
func (m *mockJobRepo) CountByStatus(id.Int, string, domainJob.Status) (int64, error) {
	return 0, nil
}
func (m *mockJobRepo) ClaimPending(id.Int, string, string, int, time.Time) ([]domainJob.Job, error) {
	return nil, nil
}
func (m *mockJobRepo) ReclaimExpired(now time.Time, limit int, defaultVisibility time.Duration) (int64, error) {
	vis := defaultVisibility
	if vis <= 0 {
		vis = 30 * time.Second
	}
	var kept []domainJob.Job
	var n int64
	for _, j := range m.running {
		if limit > 0 && n >= int64(limit) {
			kept = append(kept, j)
			continue
		}
		if j.StartedAt != nil && j.StartedAt.Add(vis).Before(now) {
			j.MarkReclaimed()
			n++
			continue
		}
		kept = append(kept, j)
	}
	m.running = kept
	return n, nil
}
func (m *mockJobRepo) ExtendLease(_ id.Int, workerID string, now time.Time) (int64, error) {
	var n int64
	for i := range m.running {
		if m.running[i].WorkerID == workerID && m.running[i].Status == domainJob.StatusRunning {
			started := now
			m.running[i].StartedAt = &started
			m.running[i].UpdatedAt = now
			n++
		}
	}
	return n, nil
}

type nopLogger struct{}

var _ logger.Service = nopLogger{}

func (nopLogger) Debug(logType.Component, string, ...interface{}) {}
func (nopLogger) Info(logType.Component, string, ...interface{})  {}
func (nopLogger) Warn(logType.Component, string, ...interface{})  {}
func (nopLogger) Error(logType.Component, string, ...interface{}) {}
func (nopLogger) Fatal(logType.Component, string, ...interface{}) {}

func TestHeartbeat_ExtendsLease(t *testing.T) {
	workerRepo := &mockWorkerRepo{}
	w, err := domainWorker.Create(domainWorker.CreateParams{
		ProjectID: 1,
		Name:      "w1",
		Queues:    []string{"q"},
	})
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}
	_ = workerRepo.Create(w)

	oldStart := clock.Now().Add(-25 * time.Second)
	job, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	job.MarkRunning(string(w.ID))
	job.StartedAt = &oldStart

	jobs := &mockJobRepo{running: []domainJob.Job{*job}}
	svc := NewService(workerRepo, jobs, nopLogger{}, nil)

	before := clock.Now()
	_, err = svc.Heartbeat(1, w.ID)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if jobs.running[0].StartedAt == nil || jobs.running[0].StartedAt.Before(before) {
		t.Fatalf("expected lease extended to >= %v, got %v", before, jobs.running[0].StartedAt)
	}
}

func TestNoHeartbeat_ReclaimThenSecondClaim(t *testing.T) {
	oldStart := clock.Now().Add(-2 * time.Minute)
	job, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	job.MarkRunning("dead-worker")
	job.StartedAt = &oldStart

	jobs := &mockJobRepo{running: []domainJob.Job{*job}}
	n, err := jobs.ReclaimExpired(clock.Now(), 500, 30*time.Second)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 reclaimed, got %d", n)
	}
	if len(jobs.running) != 0 {
		t.Fatalf("expected no running jobs, got %d", len(jobs.running))
	}

	// After reclaim the job is pending again (MarkReclaimed). Second claimer wins.
	reclaimed := job
	reclaimed.MarkReclaimed()
	if reclaimed.Status != domainJob.StatusPending {
		t.Fatalf("expected pending after reclaim, got %s", reclaimed.Status)
	}
	reclaimed.MarkRunning("new-worker")
	if reclaimed.WorkerID != "new-worker" {
		t.Fatalf("expected new-worker, got %s", reclaimed.WorkerID)
	}
}

func TestHeartbeat_WithLeaseKeepsJobFromReclaim(t *testing.T) {
	workerRepo := &mockWorkerRepo{}
	w, err := domainWorker.Create(domainWorker.CreateParams{
		ProjectID: 1,
		Name:      "w1",
		Queues:    []string{"q"},
	})
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}
	_ = workerRepo.Create(w)

	oldStart := clock.Now().Add(-25 * time.Second)
	job, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	job.MarkRunning(string(w.ID))
	job.StartedAt = &oldStart

	jobs := &mockJobRepo{running: []domainJob.Job{*job}}
	svc := NewService(workerRepo, jobs, nopLogger{}, nil)

	if _, err := svc.Heartbeat(1, w.ID); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	// Immediately after heartbeat, lease is fresh — reclaim should keep the job.
	n, err := jobs.ReclaimExpired(clock.Now(), 500, 30*time.Second)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if n != 0 || len(jobs.running) != 1 {
		t.Fatalf("expected job retained after heartbeat, reclaimed=%d running=%d", n, len(jobs.running))
	}
}
