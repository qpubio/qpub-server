package platform

import (
	"context"
	"time"

	domainCleanup "github.com/qpubio/qpub-server/internal/domain/queue/cleanup"
	queueMaintenance "github.com/qpubio/qpub-server/internal/application/service/queue/maintenance"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	taskType "github.com/qpubio/qpub-server/internal/shared/type/task"
)

// RegisterCleanupTasks registers platform maintenance tasks for queue cleanup.
func RegisterCleanupTasks(
	reg *Registry,
	cleanup domainCleanup.Service,
	maintenance *queueMaintenance.Service,
	logger logger.Service,
) error {
	tasks := []TaskDefinition{
		{
			Name:              taskType.TaskQueueWorkerCleanupMinutely,
			Schedule:          "* * * * *",
			LockTimeout:       30 * time.Second,
			IdempotencyBucket: BucketMinute,
			Handler: func(ctx context.Context, _ []byte) error {
				n, err := maintenance.PurgeStaleWorkers(ctx)
				if err != nil {
					return err
				}
				if n > 0 {
					logger.Info(log.Queue, "Purged stale workers count=%d", n)
				}
				return nil
			},
		},
		{
			Name:              taskType.TaskQueueCascadeMinutely,
			Schedule:          "* * * * *",
			LockTimeout:       2 * time.Minute,
			IdempotencyBucket: BucketMinute,
			Handler: func(ctx context.Context, _ []byte) error {
				return maintenance.ResumePendingCascades(ctx)
			},
		},
		{
			Name:              taskType.TaskQueueJobCleanupDaily,
			Schedule:          "30 0 * * *",
			LockTimeout:       5 * time.Minute,
			IdempotencyBucket: BucketDay,
			Handler: func(ctx context.Context, _ []byte) error {
				return cleanup.PurgeTerminalJobs(ctx)
			},
		},
	}
	for _, def := range tasks {
		if err := reg.Register(def); err != nil {
			return err
		}
	}
	return nil
}
