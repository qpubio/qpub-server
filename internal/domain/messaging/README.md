# Messaging System

Domain layer for real-time messaging.

## Architecture

```
Transport (WS/REST)
    → publication.Service (thin facade)
    → router.Service (publish, fanout, rate limits)
    → session.Service + egress.Pipeline (outbound delivery)
    → connection.Service (raw socket write)

NATS broker ← router (cross-instance fan-in/fan-out)
Telemetry ← router + egress events → snapshot → Redis
Event bus ← lifecycle events (connect, subscribe, channel empty)
```

## Domain packages (`internal/domain/messaging/`)

| Package         | Purpose                                                  |
| --------------- | -------------------------------------------------------- |
| `router/`       | Publish → NATS → fanout ownership                        |
| `session/`      | Connection registry, subscription cache, egress registry |
| `delivery/`     | `Deliverer` interface implemented by session             |
| `broker/`       | Broker port for cross-instance messaging                 |
| `channel/`      | Channel aggregate                                        |
| `client/`       | Client identity                                          |
| `connection/`   | Connection aggregate                                     |
| `subscription/` | Subscription aggregate                                   |
| `publication/`  | Publish ports / results                                  |
| `protocol/`     | Wire actions, JSON shapes, error codes                   |
| `envelope/`     | Message envelope with direction and source               |
| `receipt/`      | Ingress ACK/NACK and egress outcomes                     |
| `event/`        | Lifecycle events (not used for message stats)            |
| `telemetry/`    | Event types and counter model                            |
| `backpressure/` | Pressure points, drop reasons, rate limit primitives     |
| `testsupport/`  | Test helpers                                             |

Outbound egress buffering lives in the **application** layer (`application/service/messaging/egress/`), not under domain.

## Statistics

Counters flow through the messaging telemetry plane, then sync into Redis key types owned by project realtime stats (for example `msg:in`, `bw:in`, `msg:out`, `bw:out`, `msg:drop`). Gauges (`conn`, `chan`, `sub`) come from repository counts at snapshot time.

## Application services (`internal/application/service/messaging/`)

- `router/` — router implementation
- `session/` — session runtime
- `egress/` — per-connection outbound pipeline
- `telemetry/` — event subscriber + snapshot
- `backpressure/` — tenant rate limits
- `broker/`, `channel/`, `client/`, `connection/`, `subscription/`, `publication/`, `event/`

Wiring: `internal/bootstrap/modules/messaging_module.go`
