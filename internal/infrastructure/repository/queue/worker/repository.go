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

func (r *repository) ListByProject(projectID id.Int) ([]domainWorker.Worker, error) {
	var workers []domainWorker.Worker
	err := r.db.Where("project_id = ?", projectID).Order("last_seen_at DESC").Find(&workers).Error
	return workers, err
}
