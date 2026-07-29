# QPub Server

Real-time messaging (pub/sub) and product job queues. Self-hostable over REST and WebSocket.

## Features

- Pub/Sub over WebSocket + REST publish
- Product job queues (enqueue / pull / workers / webhooks)
- Tenant isolation with rate limits
- API key + client token auth
- Control API for tenants, keys, and limits
- NATS JetStream, Redis, Postgres (or CockroachDB)

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

## License

See repository license.
