package token

import (
	"github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) token.Repository {
	return &repository{db: db}
}

func (r *repository) CreateRevoke(t token.RevokedToken) (id.Int, error) {
	err := r.db.Create(&t).Error
	return t.ID, err
}

func (r *repository) PurgeExpired() error {
	return r.db.Where("expires_at < ?", clock.Now()).Delete(&token.RevokedToken{}).Error
}

func (r *repository) FindByTokenID(tokenID id.ULID) (token.RevokedToken, error) {
	var t token.RevokedToken
	err := r.db.Where("token_id = ?", tokenID).First(&t).Error
	return t, err
}
