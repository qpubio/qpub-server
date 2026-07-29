package instance

import (
	"github.com/qpubio/qpub-server/internal/domain/cluster/instance"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) instance.Repository {
	return &repository{db: db}
}

func (r *repository) Create(i *instance.ServerInstance) (id.Int, error) {
	err := r.db.Create(i).Error
	return i.ID, err
}

func (r *repository) Update(i *instance.ServerInstance) error {
	return r.db.Save(i).Error
}

func (r *repository) Delete(i *instance.ServerInstance) error {
	return r.db.Delete(i).Error
}

func (r *repository) FindByID(id id.Int) (*instance.ServerInstance, error) {
	var i instance.ServerInstance
	err := r.db.Where("id = ?", id).First(&i).Error
	return &i, err
}

func (r *repository) FindByInstanceID(instanceID id.ULID) (*instance.ServerInstance, error) {
	var i instance.ServerInstance
	err := r.db.Where("instance_id = ?", instanceID).First(&i).Error
	return &i, err
}

func (r *repository) ListActive() ([]*instance.ServerInstance, error) {
	var instances []*instance.ServerInstance
	err := r.db.Where("status = ?", instance.StatusActive).Find(&instances).Error
	return instances, err
}

func (r *repository) ListStale(threshold time.Time) ([]*instance.ServerInstance, error) {
	var instances []*instance.ServerInstance
	err := r.db.Where("status = ? AND updated_at < ?", instance.StatusActive, threshold).
		Find(&instances).Error
	return instances, err
}
