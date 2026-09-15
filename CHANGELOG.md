# Changelog

All notable changes to QPub Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.2.2] - 2026-09-15

### Fixed

- Silence `_logs` fan-out errors when no WebSocket subscribers are connected
- Skip platform tenant/queue DB lookups for sentinel IDs (project/tenant 0) to reduce GORM noise
- Make queue PUT/create idempotent on duplicate key (`idx_queue_project_name`) for concurrent bootstrap

## [v1.2.1] - 2026-09-14

### Fixed

- Batch `terminal_at` backfill in migration `2026080102` to avoid CockroachDB statement timeouts on large `jobs` tables during deploy

## [v1.2.0] - 2026-09-14

### Features

- **Queue lifecycle & cleanup** — tenant/queue `deleting` status, async cascade delete, `jobs.terminal_at`, terminal job purge
- **Control API** — `DELETE /control/v1/tenants/:id` (async `202`), `DELETE .../queues/:name` with `?force=true`
- **Platform maintenance tasks** — minutely stale-worker purge + cascade resume; daily terminal job cleanup on `_platform.*` queues
- **DTOs** — queue `status`, job `terminal_at`, worker `stale` flag

### Fixed

- Lifecycle guard treats missing queue row as writable (fixes worker register / enqueue 500 when queue is lazy-created)
- Platform runtime executes registry handlers (fixes growing `_platform.*` pending job backlog)
- Platform cron enqueue uses per-minute idempotency keys

### Changed

- New optional env vars with defaults: `QUEUE_CLEANUP_BATCH_SIZE`, `QUEUE_WORKER_STALE_DISPLAY`, `QUEUE_WORKER_STALE_DELETE`, `QUEUE_JOB_SUCCESS_RETENTION`, `QUEUE_JOB_FAILURE_RETENTION`

## [v1.1.0] - 2026-09-04

### Features

- **Queue / worker pagination** — control API `ListQueues` and `ListWorkers` return paginated responses for efficient listing at scale

### Changed

- Removed leftover admin API routes, ports, and CORS/config from the data-plane image (control / REST / WebSocket only)
- Added LICENSE and README licensing notes

### Fixed

- Redis migrate lock key is namespaced per service so backend and server no longer share the same lock

## [v1.0.1] - 2026-08-05

### Fixed

- Persist tenant rate limits to the database (control `PUT …/limits` survived only in memory)

## [v1.0.0] - 2026-08-05

### Features

- Open-source data plane: real-time messaging (pub/sub), job queues, REST, WebSocket, and control API
- Tenant / API key / rate-limit management via control API
- Queue runtime with NATS JetStream (enqueue, claim, ack/nack, schedule, DLQ)
- Boot-time dataplane schema migrations (CockroachDB / Postgres)
- Docker image and compose for self-hosting
