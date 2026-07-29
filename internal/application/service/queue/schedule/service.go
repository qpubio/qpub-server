package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/queue/router"
	"github.com/qpubio/qpub-server/internal/application/service/queue/platform"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/infrastructure/redis"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	taskType "github.com/qpubio/qpub-server/internal/shared/type/task"

	"github.com/robfig/cron/v3"
)

const (
	keyPrefixSchedule = "qpub:queue:leader"
	leaderKeyTTL      = 30 * time.Second
	leaderHeartbeat   = 20 * time.Second
)

type Service struct {
	redis      redis.Service
	router     domainRouter.Service
	registry   *platform.Registry
	logger     logger.Service
	instanceID string

	mu             sync.RWMutex
	isLeader       bool
	cronRunning    bool
	cron           *cron.Cron
	leaderStopChan chan struct{}
	leaderWg       sync.WaitGroup
	started        bool
}

func NewService(
	redis redis.Service,
	router domainRouter.Service,
	registry *platform.Registry,
	logger logger.Service,
	instanceID id.ULID,
) *Service {
	return &Service{
		redis:          redis,
		router:         router,
		registry:       registry,
		logger:         logger,
		instanceID:     string(instanceID),
		cron:           cron.New(),
		leaderStopChan: make(chan struct{}),
	}
}

func (s *Service) Start(ctx context.Context) error {
	if s.started {
		return nil
	}

	go s.runSchedulerLoop(ctx)
	go s.runLeadershipTakeover(ctx)

	if err := s.ensureLeaderAndCron(); err != nil {
		return err
	}

	s.started = true
	return nil
}

func (s *Service) ensureLeaderAndCron() error {
	isLeader, err := s.tryBecomeLeader()
	if err != nil {
		return err
	}
	if isLeader {
		if err := s.startCron(); err != nil {
			return err
		}
		s.startLeaderHeartbeat()
		s.logger.Info(log.Queue, "Instance %s is queue schedule leader with %d platform tasks", s.instanceID, len(s.registry.All()))
	}
	return nil
}

func (s *Service) runLeadershipTakeover(ctx context.Context) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.leaderStopChan:
			return
		case <-ticker.C:
			if s.IsLeader() {
				continue
			}
			if s.isLeadershipStale() {
				if err := s.ensureLeaderAndCron(); err != nil {
					s.logger.Warn(log.Queue, "Leadership takeover failed: %v", err)
				}
			}
		}
	}
}

func (s *Service) isLeadershipStale() bool {
	ttl, err := s.redis.TTL(keyPrefixSchedule).Result()
	if err != nil || ttl <= 0 {
		return true
	}
	return ttl < 5*time.Second
}

func (s *Service) Stop(ctx context.Context) error {
	close(s.leaderStopChan)
	s.leaderWg.Wait()
	if s.cron != nil {
		s.cron.Stop()
	}
	if s.isLeader {
		s.releaseLeadership()
	}
	return nil
}

func (s *Service) IsLeader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLeader
}

func (s *Service) runSchedulerLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.router.PublishDueScheduled(ctx)
			if s.IsLeader() {
				if _, err := s.router.ReclaimExpired(ctx, 500); err != nil {
					s.logger.Warn(log.Queue, "Failed to reclaim expired jobs: %v", err)
				}
			}
		case <-s.leaderStopChan:
			return
		}
	}
}

func (s *Service) startCron() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cronRunning {
		return nil
	}

	for _, def := range s.registry.All() {
		def := def
		schedule := def.Schedule
		if schedule == "" {
			continue
		}

		_, err := s.cron.AddFunc(schedule, func() {
			s.enqueuePlatformTask(def)
		})
		if err != nil {
			return fmt.Errorf("failed to schedule %s: %w", def.Name, err)
		}
		s.logger.Info(log.Queue, "Scheduled platform task task=%s schedule=%s", def.Name, schedule)
	}
	s.cron.Start()
	s.cronRunning = true
	return nil
}

func (s *Service) enqueuePlatformTask(def platform.TaskDefinition) {
	if !s.tryAcquireLock(def.Name, def.LockTimeout) {
		return
	}

	ctx := context.Background()
	queueName := platform.QueueName(def.Name)
	payload, _ := json.Marshal(map[string]string{"task": string(def.Name)})

	_, job, err := s.router.Enqueue(ctx, domainJob.EnqueueRequest{
		ProjectID: platform.PlatformProjectID,
		QueueName: queueName,
		Payload:   payload,
	})
	if err != nil {
		s.logger.Error(log.Queue, "Failed to enqueue platform task task=%s err=%v", def.Name, err)
		return
	}
	s.logger.Info(log.Queue, "Enqueued platform task task=%s job=%s queue=%s", def.Name, job.ID, queueName)
}

func (s *Service) tryAcquireLock(taskName taskType.TaskName, timeout time.Duration) bool {
	key := fmt.Sprintf("qpub:queue:execution:lock:%s", taskName)
	success, err := s.redis.SetNX(key, s.instanceID, timeout).Result()
	if err != nil {
		return false
	}
	return success
}

func (s *Service) tryBecomeLeader() (bool, error) {
	success, err := s.redis.SetNX(keyPrefixSchedule, s.instanceID, leaderKeyTTL).Result()
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	s.isLeader = success
	s.mu.Unlock()
	return success, nil
}

func (s *Service) startLeaderHeartbeat() {
	s.leaderWg.Add(1)
	go func() {
		defer s.leaderWg.Done()
		ticker := time.NewTicker(leaderHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, _ = s.redis.Set(keyPrefixSchedule, s.instanceID, leaderKeyTTL).Result()
			case <-s.leaderStopChan:
				return
			}
		}
	}()
}

func (s *Service) releaseLeadership() {
	_, _ = s.redis.Del(keyPrefixSchedule).Result()
	s.mu.Lock()
	s.isLeader = false
	s.mu.Unlock()
}

// EnqueueNow enqueues a platform task immediately.
func (s *Service) EnqueueNow(ctx context.Context, taskName taskType.TaskName) error {
	def, ok := s.registry.Get(taskName)
	if !ok {
		return fmt.Errorf("unknown platform task: %s", taskName)
	}
	s.enqueuePlatformTask(def)
	return nil
}

// noop import guard
var _ = clock.Now
