# Changelog

All notable changes to QPub Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
