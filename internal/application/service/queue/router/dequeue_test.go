package router

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
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

type allowAllGatekeeper struct{}

func (allowAllGatekeeper) AllowEnqueue(id.Int) (bool, error) { return true, nil }
func (allowAllGatekeeper) AllowDequeue(id.Int) (bool, error) { return true, nil }

type mockJobRepo struct {
	mu                sync.Mutex
	pending           []domainJob.Job
	running           []domainJob.Job
	claimCalls        atomic.Int32
	reclaimCalls      atomic.Int32
	reclaimCount      int64 // if > 0, ReclaimExpired returns this without mutating
	visibilityTimeout time.Duration
	claimDelay        time.Duration
	claimOnCall       int // return jobs only on this call number (1-based); 0 = always if pending
}

func (m *mockJobRepo) Create(*domainJob.Job) error { return nil }
func (m *mockJobRepo) Update(j *domainJob.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.running {
		if m.running[i].ID == j.ID {
			m.running[i] = *j
			return nil
		}
	}
	return nil
}
func (m *mockJobRepo) FindByID(_ id.Int, _ string, jobID id.ULID) (*domainJob.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.running {
		if m.running[i].ID == jobID {
			j := m.running[i]
			return &j, nil
		}
	}
	for i := range m.pending {
		if m.pending[i].ID == jobID {
			j := m.pending[i]
			return &j, nil
		}
	}
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

func (m *mockJobRepo) ClaimPending(projectID id.Int, queueName, workerID string, limit int, now time.Time) ([]domainJob.Job, error) {
	call := int(m.claimCalls.Add(1))
	if m.claimDelay > 0 {
		time.Sleep(m.claimDelay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.claimOnCall > 0 && call != m.claimOnCall {
		return nil, nil
	}
	if len(m.pending) == 0 || limit <= 0 {
		return nil, nil
	}
	n := limit
	if n > len(m.pending) {
		n = len(m.pending)
	}
	claimed := make([]domainJob.Job, 0, n)
	for i := 0; i < n; i++ {
		j := m.pending[i]
		j.MarkRunning(workerID)
		j.ProjectID = projectID
		j.QueueName = queueName
		_ = now
		claimed = append(claimed, j)
	}
	m.pending = m.pending[n:]
	return claimed, nil
}

func (m *mockJobRepo) ExtendLease(projectID id.Int, workerID string, now time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for i := range m.running {
		if m.running[i].WorkerID == workerID && m.running[i].Status == domainJob.StatusRunning {
			started := now
			m.running[i].StartedAt = &started
			m.running[i].UpdatedAt = now
			m.running[i].ProjectID = projectID
			n++
		}
	}
	return n, nil
}

func (m *mockJobRepo) ReclaimExpired(now time.Time, limit int, defaultVisibility time.Duration) (int64, error) {
	m.reclaimCalls.Add(1)
	if m.reclaimCount > 0 {
		return m.reclaimCount, nil
	}
	vis := m.visibilityTimeout
	if vis <= 0 {
		vis = defaultVisibility
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []domainJob.Job
	var n int64
	for _, j := range m.running {
		if limit > 0 && n >= int64(limit) {
			kept = append(kept, j)
			continue
		}
		if j.StartedAt != nil && j.StartedAt.Add(vis).Before(now) {
			j.MarkReclaimed()
			m.pending = append(m.pending, j)
			n++
			continue
		}
		kept = append(kept, j)
	}
	m.running = kept
	return n, nil
}

func newTestRouter(repo *mockJobRepo) *Service {
	return &Service{
		jobRepo:    repo,
		gatekeeper: allowAllGatekeeper{},
		logger:     nopLogger{},
	}
}

type nopLogger struct{}

var _ logger.Service = nopLogger{}

func (nopLogger) Debug(logType.Component, string, ...interface{}) {}
func (nopLogger) Info(logType.Component, string, ...interface{})  {}
func (nopLogger) Warn(logType.Component, string, ...interface{})  {}
func (nopLogger) Error(logType.Component, string, ...interface{}) {}
func (nopLogger) Fatal(logType.Component, string, ...interface{}) {}

func TestDequeue_WaitZeroSingleAttempt(t *testing.T) {
	repo := &mockJobRepo{}
	svc := newTestRouter(repo)

	jobs, err := svc.Dequeue(context.Background(), domainJob.DequeueRequest{
		ProjectID: 1,
		QueueName: "q",
		WorkerID:  "w1",
		BatchSize: 1,
		Wait:      0,
	})
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected empty, got %d", len(jobs))
	}
	if repo.claimCalls.Load() != 1 {
		t.Fatalf("expected 1 claim attempt, got %d", repo.claimCalls.Load())
	}
}

func TestDequeue_WaitDeadlineExitsWithoutInfiniteLoop(t *testing.T) {
	repo := &mockJobRepo{}
	svc := newTestRouter(repo)

	start := clock.Now()
	jobs, err := svc.Dequeue(context.Background(), domainJob.DequeueRequest{
		ProjectID: 1,
		QueueName: "q",
		WorkerID:  "w1",
		BatchSize: 1,
		Wait:      200 * time.Millisecond,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected empty, got %d", len(jobs))
	}
	if elapsed > 2*time.Second {
		t.Fatalf("dequeue hung too long: %v", elapsed)
	}
	if repo.claimCalls.Load() < 1 {
		t.Fatal("expected at least one claim attempt")
	}
}

func TestDequeue_ContextCancel(t *testing.T) {
	repo := &mockJobRepo{claimDelay: 50 * time.Millisecond}
	svc := newTestRouter(repo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Dequeue(ctx, domainJob.DequeueRequest{
		ProjectID: 1,
		QueueName: "q",
		WorkerID:  "w1",
		Wait:      time.Second,
	})
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestDequeue_ClaimsPending(t *testing.T) {
	j, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	repo := &mockJobRepo{pending: []domainJob.Job{*j}}
	svc := newTestRouter(repo)

	jobs, err := svc.Dequeue(context.Background(), domainJob.DequeueRequest{
		ProjectID: 1,
		QueueName: "q",
		WorkerID:  "w1",
		BatchSize: 1,
		Wait:      0,
	})
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != domainJob.StatusRunning {
		t.Fatalf("expected running, got %s", jobs[0].Status)
	}
}

func TestDequeue_ConcurrentClaimsOnlyOneWins(t *testing.T) {
	j, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	repo := &mockJobRepo{pending: []domainJob.Job{*j}}
	svc := newTestRouter(repo)

	var wg sync.WaitGroup
	results := make(chan []domainJob.Job, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs, err := svc.Dequeue(context.Background(), domainJob.DequeueRequest{
				ProjectID: 1,
				QueueName: "q",
				WorkerID:  "w",
				BatchSize: 1,
				Wait:      0,
			})
			if err != nil {
				t.Errorf("dequeue: %v", err)
				results <- nil
				return
			}
			results <- jobs
		}()
	}
	wg.Wait()
	close(results)

	total := 0
	for jobs := range results {
		total += len(jobs)
	}
	if total != 1 {
		t.Fatalf("expected exactly one winner, claimed total=%d", total)
	}
}

func TestReclaimExpired_DelegatesToRepo(t *testing.T) {
	repo := &mockJobRepo{reclaimCount: 42}
	svc := newTestRouter(repo)
	n, err := svc.ReclaimExpired(context.Background(), 100)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}
}

func TestReclaimExpired_MovesExpiredRunningToPending(t *testing.T) {
	j, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	j.MarkRunning("w1")
	started := clock.Now().Add(-2 * time.Minute)
	j.StartedAt = &started

	fresh, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	fresh.MarkRunning("w2")

	repo := &mockJobRepo{
		running:           []domainJob.Job{*j, *fresh},
		visibilityTimeout: 30 * time.Second,
	}
	svc := newTestRouter(repo)

	n, err := svc.ReclaimExpired(context.Background(), 500)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 reclaimed, got %d", n)
	}
	if len(repo.pending) != 1 || repo.pending[0].Status != domainJob.StatusPending {
		t.Fatalf("expected expired job back in pending, got %+v", repo.pending)
	}
	if len(repo.running) != 1 || repo.running[0].WorkerID != "w2" {
		t.Fatalf("expected fresh running job kept, got %+v", repo.running)
	}
	_ = domainQueue.DefaultVisibilityTimeout
}

func TestAck_WorkerMismatchRejected(t *testing.T) {
	j, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	j.MarkRunning("owner-worker")
	repo := &mockJobRepo{running: []domainJob.Job{*j}}
	svc := newTestRouter(repo)

	_, err := svc.Ack(context.Background(), domainJob.AckRequest{
		ProjectID: 1,
		QueueName: "q",
		JobID:     j.ID,
		WorkerID:  "other-worker",
	})
	if err != domainJob.ErrWorkerMismatch {
		t.Fatalf("expected ErrWorkerMismatch, got %v", err)
	}
}

func TestNack_WorkerMismatchRejected(t *testing.T) {
	j, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	j.MarkRunning("owner-worker")
	repo := &mockJobRepo{running: []domainJob.Job{*j}}
	svc := newTestRouter(repo)

	_, err := svc.Nack(context.Background(), domainJob.NackRequest{
		ProjectID: 1,
		QueueName: "q",
		JobID:     j.ID,
		WorkerID:  "other-worker",
		Reason:    "nope",
	})
	if err != domainJob.ErrWorkerMismatch {
		t.Fatalf("expected ErrWorkerMismatch, got %v", err)
	}
}

func TestDequeue_TouchesRegisteredWorker(t *testing.T) {
	j, _ := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID: 1,
		QueueName: "q",
		Payload:   []byte(`{}`),
	})
	repo := &mockJobRepo{pending: []domainJob.Job{*j}}
	workers := &mockWorkerSvc{}
	svc := newTestRouter(repo)
	svc.workerSvc = workers

	jobs, err := svc.Dequeue(context.Background(), domainJob.DequeueRequest{
		ProjectID: 1,
		QueueName: "q",
		WorkerID:  "registered-1",
		BatchSize: 1,
		Wait:      0,
	})
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if workers.heartbeatCalls.Load() != 1 {
		t.Fatalf("expected heartbeat touch, got %d", workers.heartbeatCalls.Load())
	}
}

type mockWorkerSvc struct {
	heartbeatCalls atomic.Int32
}

func (m *mockWorkerSvc) Register(domainWorker.CreateParams) (domainWorker.Worker, error) {
	return domainWorker.Worker{}, nil
}
func (m *mockWorkerSvc) Heartbeat(id.Int, id.ULID) (domainWorker.Worker, error) {
	m.heartbeatCalls.Add(1)
	return domainWorker.Worker{}, nil
}
func (m *mockWorkerSvc) Get(id.Int, id.ULID) (domainWorker.Worker, error) {
	return domainWorker.Worker{}, domainWorker.ErrNotFound
}
func (m *mockWorkerSvc) ListByProject(id.Int) ([]domainWorker.Worker, error) {
	return nil, nil
}
