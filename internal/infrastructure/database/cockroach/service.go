package cockroach

import (
	"fmt"
	infrastructure "github.com/qpubio/qpub-server/internal/config/infrastructure"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"strings"
	"time"

	"github.com/lib/pq"
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
	// Build DSN with password (CockroachDB in insecure mode ignores password but accepts it)
	var dsn string
	if s.cfg.Password != "" {
		dsn = fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s?sslmode=%s&timezone=%s",
			s.cfg.User,
			s.cfg.Password,
			s.cfg.Host,
			s.cfg.Port,
			s.cfg.Name,
			s.cfg.Options.SSLMode,
			s.cfg.Options.Timezone,
		)
	} else {
		dsn = fmt.Sprintf(
			"postgresql://%s@%s:%s/%s?sslmode=%s&timezone=%s",
			s.cfg.User,
			s.cfg.Host,
			s.cfg.Port,
			s.cfg.Name,
			s.cfg.Options.SSLMode,
			s.cfg.Options.Timezone,
		)
	}

	// Add SSL certificate paths if SSLMode requires them
	if s.cfg.Options.SSLMode != "disable" {
		if s.cfg.Options.SSLRootCert != "" {
			dsn += fmt.Sprintf("&sslrootcert=%s", s.cfg.Options.SSLRootCert)
		}
		if s.cfg.Options.SSLCert != "" {
			dsn += fmt.Sprintf("&sslcert=%s", s.cfg.Options.SSLCert)
		}
		if s.cfg.Options.SSLKey != "" {
			dsn += fmt.Sprintf("&sslkey=%s", s.cfg.Options.SSLKey)
		}
	}

	config := &gorm.Config{
		PrepareStmt: true,
		NowFunc: func() time.Time {
			return clock.Now()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cockroachdb: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB instance: %w", err)
	}

	// Configure connection pool
	// CockroachDB recommends lower connection counts than PostgreSQL
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping cockroachdb: %w", err)
	}

	// Add CockroachDB specific middleware for retries
	db.Use(&cockroachRetryPlugin{
		maxRetries: s.cfg.Cockroach.Retries,
		logger:     s.logger,
	})

	s.db = db
	s.logger.Info(log.DB, "Connected to CockroachDB database")
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

// Retry plugin for CockroachDB
type cockroachRetryPlugin struct {
	maxRetries int
	logger     logger.Service
}

func (p *cockroachRetryPlugin) Name() string {
	return "cockroachRetryPlugin"
}

func (p *cockroachRetryPlugin) Initialize(db *gorm.DB) error {
	// Register callbacks after operations to check for retry errors
	callback := p.createRetryCallback()
	db.Callback().Create().After("gorm:create").Register("cockroach:retry_create", callback)
	db.Callback().Update().After("gorm:update").Register("cockroach:retry_update", callback)
	db.Callback().Delete().After("gorm:delete").Register("cockroach:retry_delete", callback)
	db.Callback().Query().After("gorm:query").Register("cockroach:retry_query", callback)
	db.Callback().Row().After("gorm:row").Register("cockroach:retry_row", callback)
	db.Callback().Raw().After("gorm:raw").Register("cockroach:retry_raw", callback)
	return nil
}

func (p *cockroachRetryPlugin) createRetryCallback() func(*gorm.DB) {
	return func(db *gorm.DB) {
		// Only handle errors, don't retry automatically as GORM doesn't support
		// replaying operations. The application should use transactions with
		// proper retry logic for CockroachDB.
		if db.Error != nil && isRetryableError(db.Error) {
			p.logger.Warn(log.DB, "CockroachDB retryable error detected: %v (hint: wrap in transaction with retry logic)", db.Error)
		}
	}
}

// isRetryableError checks if the error is a CockroachDB serialization error
// that should be retried at the transaction level
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for PostgreSQL error codes that CockroachDB uses
	if pqErr, ok := err.(*pq.Error); ok {
		// 40001 = serialization_failure (retry transaction)
		// 40003 = statement_completion_unknown
		// CR000 = CockroachDB specific retry error
		code := string(pqErr.Code)
		if code == "40001" || code == "40003" || strings.HasPrefix(code, "CR") {
			return true
		}
	}

	// Check error message for CockroachDB retry hints
	errMsg := err.Error()
	return strings.Contains(errMsg, "restart transaction") ||
		strings.Contains(errMsg, "retry transaction") ||
		strings.Contains(errMsg, "TransactionRetryWithProtoRefreshError") ||
		strings.Contains(errMsg, "RETRY_SERIALIZABLE")
}

// ExecuteWithRetry wraps a transaction with automatic retry logic for CockroachDB
// This is a helper function that applications can use for critical operations
func ExecuteWithRetry(db *gorm.DB, maxRetries int, logger logger.Service, fn func(*gorm.DB) error) error {
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = db.Transaction(func(tx *gorm.DB) error {
			return fn(tx)
		})

		if err == nil {
			return nil
		}

		if !isRetryableError(err) {
			return err
		}

		if attempt < maxRetries {
			backoff := time.Duration(attempt+1) * 100 * time.Millisecond
			logger.Warn(log.DB, "Retrying CockroachDB transaction (attempt %d/%d) after %v: %v",
				attempt+1, maxRetries, backoff, err)
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("transaction failed after %d retries: %w", maxRetries, err)
}
