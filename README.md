# QPub Server

Open-source **data-plane** for real-time messaging (pub/sub) and product queues.

This binary does **not** include billing, accounts, dashboard auth, Stripe, or OAuth. Provision tenants, API keys, and rate limits via the control API (or a separate control plane).

## Features

- Pub/Sub over WebSocket + REST publish
- Product job queues (enqueue / pull / workers / webhooks)
- Tenant isolation with pushed rate limits
- API key + client token auth
- Control API for tenants / keys / limits
- NATS JetStream, Redis, Postgres

## Quick start

```bash
cp .env.example .env
docker compose up --build
```

Ports (defaults):

| Service   | Port |
|-----------|------|
| Admin     | 8081 |
| Control   | 8091 |
| REST      | 8111 |
| WebSocket | 8131 |

## Local build

```bash
go mod tidy
go build -o bin/qpub-server ./cmd/server
./bin/qpub-server
```

Requires Postgres, Redis, and NATS (JetStream) reachable via `.env`.

## Control API

Set `CONTROL_API_TOKEN` and call (Bearer or `X-Control-Token`):

- `POST /control/v1/tenants`
- `PUT /control/v1/tenants/:id/limits`
- `POST /control/v1/tenants/:id/keys`

## OSS boundary

```bash
./scripts/check-oss-boundary.sh
```

Fails if `billing`, `domain/account`, `domain/user`, `stripe`, or `oauth` appear under `internal/`.

## License

See repository license.
