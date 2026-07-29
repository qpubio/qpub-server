package apikey

import (
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) apikey.Repository {
	return &repository{db: db}
}

func (r *repository) Create(ak *apikey.APIKey) (id.Int, error) {
	err := r.db.Create(&ak).Error
	return ak.ID, err
}

func (r *repository) Update(ak *apikey.APIKey) error {
	return r.db.Save(&ak).Error
}

func (r *repository) Delete(ak *apikey.APIKey) error {
	return r.db.Delete(&ak).Error
}

func (r *repository) FindByID(akID id.Int) (*apikey.APIKey, error) {
	var ak apikey.APIKey
	err := r.db.Where("id = ?", akID).First(&ak).Error
	return &ak, err
}

func (r *repository) FindByIDs(akIDs []id.Int) ([]apikey.APIKey, error) {
	var apiKeys []apikey.APIKey
	err := r.db.Where("id IN (?)", akIDs).Find(&apiKeys).Error
	return apiKeys, err
}

func (r *repository) ListByProjectID(projID id.Int) ([]apikey.APIKey, error) {
	var apiKeys []apikey.APIKey
	err := r.db.Where("project_id = ?", projID).Find(&apiKeys).Error
	return apiKeys, err
}
