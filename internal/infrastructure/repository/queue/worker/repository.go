package worker

import (
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
