package tenant

import (
	"errors"

	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) tenant.Repository {
	return &repository{db: db}
}

func (r *repository) UpsertTenant(t tenant.Tenant) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&t).Error
}

func (r *repository) UpdateTenant(t tenant.Tenant) error {
	return r.db.Save(&t).Error
}

func (r *repository) DeleteTenant(tenantID id.Int) error {
	return r.db.Where("id = ?", tenantID).Delete(&tenant.Tenant{}).Error
}

func (r *repository) ListDeleting(limit int) ([]tenant.Tenant, error) {
	if limit <= 0 {
		limit = 100
	}
	var tenants []tenant.Tenant
	err := r.db.Where("status = ?", tenant.StatusDeleting).
		Order("updated_at ASC").
		Limit(limit).
		Find(&tenants).Error
	return tenants, err
}

func (r *repository) FindTenant(tenantID id.Int) (*tenant.Tenant, error) {
	var t tenant.Tenant
	err := r.db.Where("id = ?", tenantID).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *repository) UpsertLimits(l tenant.Limits) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		UpdateAll: true,
	}).Create(&l).Error
}

func (r *repository) FindLimits(tenantID id.Int) (*tenant.Limits, error) {
	var l tenant.Limits
	err := r.db.Where("tenant_id = ?", tenantID).First(&l).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *repository) DeleteLimits(tenantID id.Int) error {
	return r.db.Where("tenant_id = ?", tenantID).Delete(&tenant.Limits{}).Error
}
