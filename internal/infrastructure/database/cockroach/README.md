# CockroachDB Support

This package provides CockroachDB database support for the application using GORM. CockroachDB is a distributed SQL database that is compatible with PostgreSQL but requires special handling for transaction retries.

## Configuration

CockroachDB is configured through environment variables in the `.env` file:

```bash
# Database Configuration
DB_DRIVER=cockroach              # Set to "cockroach" to use CockroachDB
DB_HOST=cockroach               # CockroachDB host
DB_PORT=26257                   # CockroachDB default port
DB_USER=root                    # CockroachDB user
DB_PASSWORD=                    # CockroachDB password (empty for insecure mode)
DB_NAME=qpub_backend            # Database name
DB_SSLMODE=disable             # SSL mode (use "require" for production)
DB_TIMEZONE=UTC                # Timezone
DB_COCKROACH_RETRIES=3         # Number of retry attempts for transactions
```

## Docker Compose Setup

To use CockroachDB in docker-compose:

1. Comment out the `postgres` service in `docker-compose.yml`
2. Uncomment the `cockroach` service
3. Update `.env` with the CockroachDB configuration above
4. Run `docker-compose up -d`

CockroachDB provides a web UI at `http://localhost:8080` for monitoring and administration.

## Transaction Retries

CockroachDB uses optimistic concurrency control and may return serialization errors that require transaction retries. This package provides several ways to handle retries:

### 1. Automatic Error Detection (Passive)

The CockroachDB service includes a plugin that detects retryable errors and logs warnings. This helps identify operations that should be wrapped in retry logic:

```go
// The plugin will log warnings for retryable errors:
// "CockroachDB retryable error detected: ... (hint: wrap in transaction with retry logic)"
```

### 2. Using ExecuteWithRetry Helper (Recommended)

For critical operations, use the `ExecuteWithRetry` helper function:

```go
import (
    "qpub/internal/infrastructure/database/cockroach"
)

// In your service or repository
err := cockroach.ExecuteWithRetry(db, 3, logger, func(tx *gorm.DB) error {
    // Your transactional operations here
    if err := tx.Create(&user).Error; err != nil {
        return err
    }

    if err := tx.Create(&account).Error; err != nil {
        return err
    }

    return nil
})
```

### 3. Manual Retry Logic

For custom retry strategies, implement your own retry loop:

```go
var err error
for attempt := 0; attempt < maxRetries; attempt++ {
    err = db.Transaction(func(tx *gorm.DB) error {
        // Your operations
        return nil
    })

    if err == nil {
        break
    }

    // Check if error is retryable using the exported function
    if !isRetryableError(err) {
        return err
    }

    time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
}
```

## Connection Pool Configuration

CockroachDB connection pool is configured with recommended settings:

- **MaxIdleConns**: 5 (lower than PostgreSQL due to distributed nature)
- **MaxOpenConns**: 50 (conservative limit for distributed transactions)
- **ConnMaxLifetime**: 1 hour

These settings are optimized for CockroachDB's distributed architecture and can be adjusted in the `service.go` file if needed.

## PostgreSQL Compatibility

CockroachDB is largely compatible with PostgreSQL, which means:

- ✅ GORM operations work without modification
- ✅ PostgreSQL wire protocol is used
- ✅ Most SQL queries are compatible
- ⚠️ Some PostgreSQL-specific features may not be available
- ⚠️ Transaction retry logic is required for high-contention scenarios

## Error Codes

CockroachDB uses PostgreSQL error codes for serialization failures:

- `40001`: Serialization failure (retry transaction)
- `40003`: Statement completion unknown
- `CR***`: CockroachDB-specific error codes

The package also checks for error messages containing:

- "restart transaction"
- "retry transaction"
- "TransactionRetryWithProtoRefreshError"
- "RETRY_SERIALIZABLE"

## Best Practices

1. **Always wrap critical multi-statement operations in transactions** with retry logic
2. **Use short transactions** to minimize contention
3. **Avoid long-running transactions** that can cause serialization conflicts
4. **Monitor retry logs** to identify hot spots in your application
5. **Use batch operations** when possible to reduce transaction count
6. **Test with realistic load** to identify serialization issues early

## Differences from PostgreSQL

While CockroachDB is PostgreSQL-compatible, there are some differences:

1. **Transaction Retries**: Required for high-concurrency scenarios
2. **Connection Pool**: Smaller pool size recommended
3. **Serial Data Type**: CockroachDB handles sequences differently
4. **Foreign Keys**: Some foreign key behaviors differ
5. **Locking**: Optimistic locking instead of pessimistic

## Migration

Migrations are handled by GORM's AutoMigrate, which works with both PostgreSQL and CockroachDB. The same migration files are used for both databases.

## Monitoring

CockroachDB provides:

- Web UI at `http://localhost:8080` (when using docker-compose)
- Built-in metrics and monitoring
- SQL query execution plans
- Distributed trace information

## References

- [CockroachDB Documentation](https://www.cockroachlabs.com/docs/)
- [Transaction Retry Logic](https://www.cockroachlabs.com/docs/stable/transaction-retry-error-reference.html)
- [PostgreSQL Compatibility](https://www.cockroachlabs.com/docs/stable/postgresql-compatibility.html)
