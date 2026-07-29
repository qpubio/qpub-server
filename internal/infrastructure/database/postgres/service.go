package postgres

import (
	"fmt"
	infrastructure "github.com/qpubio/qpub-server/internal/config/infrastructure"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type service struct {
	cfg    *infrastructure.Database
	logger logger.Service
	db     *gorm.DB
}

func NewService(cfg *infrastructure.Database, logger logger.Service) (*service, error) {
	return &service{
		cfg:    cfg,
		logger: logger,
	}, nil
}

func (s *service) Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		s.cfg.Host,
		s.cfg.User,
		s.cfg.Password,
		s.cfg.Name,
		s.cfg.Port,
		s.cfg.Options.SSLMode,
		s.cfg.Options.Timezone,
	)

	// Add SSL certificate paths if SSLMode requires them
	if s.cfg.Options.SSLMode != "disable" {
		if s.cfg.Options.SSLRootCert != "" {
			dsn += fmt.Sprintf(" sslrootcert=%s", s.cfg.Options.SSLRootCert)
		}
		if s.cfg.Options.SSLCert != "" {
			dsn += fmt.Sprintf(" sslcert=%s", s.cfg.Options.SSLCert)
		}
		if s.cfg.Options.SSLKey != "" {
			dsn += fmt.Sprintf(" sslkey=%s", s.cfg.Options.SSLKey)
		}
	}

	if s.cfg.Postgres.SearchPath != "" {
		dsn += fmt.Sprintf(" search_path=%s", s.cfg.Postgres.SearchPath)
	}

	config := &gorm.Config{
		PrepareStmt: true,
		NowFunc: func() time.Time {
			return clock.Now()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB instance: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	s.db = db
	s.logger.Info(log.DB, "Connected to PostgreSQL database")
	return db, nil
}

func (s *service) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			return fmt.Errorf("failed to get sql.DB instance: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}

func (s *service) DB() *gorm.DB {
	return s.db
}
