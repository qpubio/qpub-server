# QPub Server

Real-time messaging (pub/sub) and job queues. Self-hostable over REST and WebSocket.

## Features

- Pub/Sub over WebSocket + REST publish
- Job queues (enqueue / pull / workers / webhooks)
- Tenant isolation with rate limits
- API key + client token auth
- Control API for tenants, keys, limits, and queue admin
- NATS JetStream, Redis, Postgres (or CockroachDB)

## Quick start

```bash
cp .env.example .env
docker compose up --build
```

Compose uses **Postgres** (plus Redis and NATS). For CockroachDB outside this compose stack, set `DB_DRIVER=cockroach` (see `.env.example`).

Ports (defaults):

| Service              | Port | Notes                          |
|----------------------|------|--------------------------------|
| Control              | 8091 | Provisioning API               |
| REST                 | 8111 | Publish, queues, tokens        |
| WebSocket            | 8131 | Real-time messaging            |

Create a tenant (and an API key) via the Control API before calling REST or WebSocket.

## Local build

```bash
go mod tidy
go build -o bin/qpub-server ./cmd/server
./bin/qpub-server
```

Requires Postgres (or Cockroach), Redis, and NATS (JetStream) reachable via `.env`.

## Control API

Authenticate with `CONTROL_API_TOKEN` (`Authorization: Bearer …` or `X-Control-Token`). If the token is empty, the control API is open (local dev only).

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/health` | Health (no auth) |
| POST | `/control/v1/tenants` | Ensure tenant `{ "id": <int> }` |
| GET | `/control/v1/tenants/:tenantID` | Get tenant |
| DELETE | `/control/v1/tenants/:tenantID` | Delete tenant and its keys |
| PUT | `/control/v1/tenants/:tenantID/limits` | Set rate limits |
| GET | `/control/v1/tenants/:tenantID/limits` | Get rate limits |
| POST | `/control/v1/tenants/:tenantID/keys` | Create API key |
| GET | `/control/v1/tenants/:tenantID/keys` | List API keys |
| DELETE | `/control/v1/tenants/:tenantID/keys/:keyID` | Delete API key |
| GET | `/control/v1/tenants/:tenantID/queues` | List queues |
| GET | `/control/v1/tenants/:tenantID/workers` | List workers |
| GET | `/control/v1/metrics` | Metrics export hook |
