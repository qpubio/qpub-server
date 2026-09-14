package worker

import (
	"time"

	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domainWorker.Repository {
	return &repository{db: db}
}

func (r *repository) Create(worker *domainWorker.Worker) error {
	return r.db.Create(worker).Error
}

func (r *repository) Update(worker *domainWorker.Worker) error {
	return r.db.Save(worker).Error
}

func (r *repository) FindByID(projectID id.Int, workerID id.ULID) (*domainWorker.Worker, error) {
	var w domainWorker.Worker
	err := r.db.Where("project_id = ? AND id = ?", projectID, workerID).First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *repository) ListByProjectPaginated(projectID id.Int, limit, offset int) ([]domainWorker.Worker, int64, error) {
	var total int64
	if err := r.db.Model(&domainWorker.Worker{}).
		Where("project_id = ?", projectID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var workers []domainWorker.Worker
	err := r.db.Where("project_id = ?", projectID).
		Order("last_seen_at DESC").
		Limit(limit).Offset(offset).
		Find(&workers).Error
	return workers, total, err
}

func (r *repository) Delete(projectID id.Int, workerID id.ULID) error {
	return r.db.Where("project_id = ? AND id = ?", projectID, workerID).Delete(&domainWorker.Worker{}).Error
}

func (r *repository) ListByProject(projectID id.Int) ([]domainWorker.Worker, error) {
	var workers []domainWorker.Worker
	err := r.db.Where("project_id = ?", projectID).Find(&workers).Error
	return workers, err
}

func (r *repository) ListStale(before time.Time, limit int) ([]domainWorker.Worker, error) {
	if limit <= 0 {
		limit = 500
	}
	var workers []domainWorker.Worker
	err := r.db.Where("last_seen_at < ?", before).
		Order("last_seen_at ASC").
		Limit(limit).
		Find(&workers).Error
	return workers, err
}

func (r *repository) HasRunningJobs(projectID id.Int, workerID string) (bool, error) {
	var count int64
	err := r.db.Table("jobs").
		Where("project_id = ? AND worker_id = ? AND status = ?", projectID, workerID, "running").
		Count(&count).Error
	return count > 0, err
}
