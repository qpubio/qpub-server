package job

import (
	"encoding/json"
	"time"

	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domainJob.Repository {
	return &repository{db: db}
}

func (r *repository) Create(job *domainJob.Job) error {
	return r.db.Create(job).Error
}

func (r *repository) Update(job *domainJob.Job) error {
	return r.db.Save(job).Error
}

func (r *repository) FindByID(projectID id.Int, queueName string, jobID id.ULID) (*domainJob.Job, error) {
	var j domainJob.Job
	err := r.db.Where("project_id = ? AND queue_name = ? AND id = ?", projectID, queueName, jobID).First(&j).Error
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *repository) FindByIdempotencyKey(projectID id.Int, queueName, key string) (*domainJob.Job, error) {
	if key == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var j domainJob.Job
	err := r.db.Where("project_id = ? AND queue_name = ? AND idempotency_key = ?", projectID, queueName, key).First(&j).Error
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *repository) List(filter domainJob.ListFilter) ([]domainJob.Job, error) {
	query := r.db.Where("project_id = ?", filter.ProjectID)
	if filter.QueueName != "" {
		query = query.Where("queue_name = ?", filter.QueueName)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var jobs []domainJob.Job
	err := query.Order("created_at DESC").Limit(limit).Offset(filter.Offset).Find(&jobs).Error
	return jobs, err
}

func (r *repository) FindDueScheduled(limit int, now time.Time) ([]domainJob.Job, error) {
	if limit <= 0 {
		limit = 100
	}
	var jobs []domainJob.Job
	err := r.db.Raw(`
SELECT j.*
FROM jobs j
INNER JOIN queues q ON q.project_id = j.project_id AND q.name = j.queue_name
INNER JOIN tenants t ON t.id = j.project_id
WHERE j.status = ?
  AND j.schedule_at IS NOT NULL
  AND j.schedule_at <= ?
  AND q.status = 'active'
  AND t.status = 'active'
ORDER BY j.schedule_at ASC
LIMIT ?
`, domainJob.StatusScheduled, now, limit).Scan(&jobs).Error
	return jobs, err
}

func (r *repository) CountByStatus(projectID id.Int, queueName string, status domainJob.Status) (int64, error) {
	var count int64
	err := r.db.Model(&domainJob.Job{}).
		Where("project_id = ? AND queue_name = ? AND status = ?", projectID, queueName, status).
		Count(&count).Error
	return count, err
}

func (r *repository) ClaimPending(projectID id.Int, queueName, workerID string, limit int, now time.Time) ([]domainJob.Job, error) {
	if limit <= 0 {
		limit = 1
	}

	var jobs []domainJob.Job
	err := r.db.Raw(`
UPDATE jobs AS j
SET
	status = ?,
	worker_id = ?,
	attempt = j.attempt + 1,
	started_at = ?,
	updated_at = ?
FROM (
	SELECT id
	FROM jobs
	WHERE project_id = ?
		AND queue_name = ?
		AND status IN (?, ?)
		AND (schedule_at IS NULL OR schedule_at <= ?)
	ORDER BY schedule_at ASC NULLS FIRST, created_at ASC
	LIMIT ?
	FOR UPDATE SKIP LOCKED
) AS selected
WHERE j.id = selected.id
RETURNING j.*
`,
		domainJob.StatusRunning,
		workerID,
		now,
		now,
		projectID,
		queueName,
		domainJob.StatusPending,
		domainJob.StatusScheduled,
		now,
		limit,
	).Scan(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *repository) ReclaimExpired(now time.Time, limit int, defaultVisibility time.Duration) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	if defaultVisibility <= 0 {
		defaultVisibility = 30 * time.Second
	}

	// visibility_timeout is stored as int64 nanoseconds. Cockroach intervals have
	// no "nanosecond" unit, so convert to seconds. Scalar subquery (not LEFT JOIN)
	// so FOR UPDATE can lock jobs rows (SQLSTATE 0A000 otherwise).
	result := r.db.Exec(`
UPDATE jobs AS j
SET
	status = ?,
	worker_id = '',
	started_at = NULL,
	updated_at = ?
FROM (
	SELECT j2.id
	FROM jobs j2
	INNER JOIN queues q ON q.project_id = j2.project_id AND q.name = j2.queue_name
	INNER JOIN tenants t ON t.id = j2.project_id
	WHERE j2.status = ?
		AND q.status = 'active'
		AND t.status = 'active'
		AND j2.started_at IS NOT NULL
		AND j2.started_at + (
			(
				COALESCE(
					(
						SELECT q.visibility_timeout
						FROM queues q
						WHERE q.project_id = j2.project_id AND q.name = j2.queue_name
						LIMIT 1
					),
					?
				)::FLOAT8 / 1000000000.0
			) * INTERVAL '1 second'
		) < ?
	ORDER BY j2.started_at ASC
	LIMIT ?
	FOR UPDATE SKIP LOCKED
) AS selected
WHERE j.id = selected.id
`,
		domainJob.StatusPending,
		now,
		domainJob.StatusRunning,
		int64(defaultVisibility),
		now,
		limit,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *repository) ExtendLease(projectID id.Int, workerID string, now time.Time) (int64, error) {
	if workerID == "" {
		return 0, nil
	}
	result := r.db.Model(&domainJob.Job{}).
		Where("project_id = ? AND worker_id = ? AND status = ?", projectID, workerID, domainJob.StatusRunning).
		Updates(map[string]interface{}{
			"started_at": now,
			"updated_at": now,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *repository) UpdateMetadata(projectID id.Int, queueName string, jobID id.ULID, metadata json.RawMessage, now time.Time) error {
	return r.db.Model(&domainJob.Job{}).
		Where("project_id = ? AND queue_name = ? AND id = ?", projectID, queueName, jobID).
		Updates(map[string]interface{}{
			"metadata":   metadata,
			"updated_at": now,
		}).Error
}

func (r *repository) CountByQueue(projectID id.Int, queueName string) (int64, error) {
	var count int64
	err := r.db.Model(&domainJob.Job{}).
		Where("project_id = ? AND queue_name = ?", projectID, queueName).
		Count(&count).Error
	return count, err
}

func (r *repository) CountActiveByQueue(projectID id.Int, queueName string) (int64, error) {
	var count int64
	err := r.db.Model(&domainJob.Job{}).
		Where("project_id = ? AND queue_name = ? AND status IN ?", projectID, queueName,
			[]domainJob.Status{domainJob.StatusPending, domainJob.StatusScheduled, domainJob.StatusRunning}).
		Count(&count).Error
	return count, err
}

func (r *repository) PurgeTerminalBefore(statuses []domainJob.Status, terminalBefore time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 1000
	}
	if len(statuses) == 0 {
		return 0, nil
	}
	result := r.db.Exec(`
DELETE FROM jobs
WHERE id IN (
	SELECT id FROM jobs
	WHERE status IN ?
	  AND terminal_at IS NOT NULL
	  AND terminal_at < ?
	LIMIT ?
)
`, statuses, terminalBefore, limit)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *repository) DeleteByQueue(projectID id.Int, queueName string, limit int) (int64, error) {
	if limit <= 0 {
		limit = 1000
	}
	result := r.db.Exec(`
DELETE FROM jobs
WHERE id IN (
	SELECT id FROM jobs
	WHERE project_id = ? AND queue_name = ?
	LIMIT ?
)
`, projectID, queueName, limit)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *repository) DeleteByProject(projectID id.Int, limit int) (int64, error) {
	if limit <= 0 {
		limit = 1000
	}
	result := r.db.Exec(`
DELETE FROM jobs
WHERE id IN (
	SELECT id FROM jobs
	WHERE project_id = ?
	LIMIT ?
)
`, projectID, limit)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *repository) ForceCancelActiveByQueue(projectID id.Int, queueName string, now time.Time) (int64, error) {
	result := r.db.Exec(`
UPDATE jobs
SET status = ?, terminal_at = ?, updated_at = ?, worker_id = '', started_at = NULL
WHERE project_id = ? AND queue_name = ? AND status IN ?
`, domainJob.StatusCancelled, now, now, projectID, queueName,
		[]domainJob.Status{domainJob.StatusPending, domainJob.StatusScheduled, domainJob.StatusRunning})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
