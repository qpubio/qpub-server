package queue

import (
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domainQueue.Repository {
	return &repository{db: db}
}

func (r *repository) Create(queue *domainQueue.Queue) (id.Int, error) {
	if err := r.db.Create(queue).Error; err != nil {
		return 0, err
	}
	return queue.ID, nil
}

func (r *repository) Update(queue *domainQueue.Queue) error {
	return r.db.Save(queue).Error
}

func (r *repository) FindByProjectAndName(projectID id.Int, name string) (*domainQueue.Queue, error) {
	var q domainQueue.Queue
	err := r.db.Where("project_id = ? AND name = ?", projectID, name).First(&q).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *repository) FindByID(queueID id.Int) (*domainQueue.Queue, error) {
	var q domainQueue.Queue
	err := r.db.First(&q, queueID).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *repository) ListByProject(projectID id.Int) ([]domainQueue.Queue, error) {
	var queues []domainQueue.Queue
	err := r.db.Where("project_id = ?", projectID).Find(&queues).Error
	return queues, err
}
