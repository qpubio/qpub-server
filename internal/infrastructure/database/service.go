package database

import (
	"fmt"
	infrastructure "github.com/qpubio/qpub-server/internal/config/infrastructure"
	"github.com/qpubio/qpub-server/internal/infrastructure/database/cockroach"
	"github.com/qpubio/qpub-server/internal/infrastructure/database/postgres"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"

	"gorm.io/gorm"
)

type Service interface {
	Connect() (*gorm.DB, error)
	Close() error
	DB() *gorm.DB
}

func New(cfg *infrastructure.Database, logger logger.Service) (Service, error) {
	switch cfg.Driver {
	case "postgres":
		return postgres.NewService(cfg, logger)
	case "cockroach":
		return cockroach.NewService(cfg, logger)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}
