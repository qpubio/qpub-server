# Messaging System

Domain layer for real-time messaging. See [docs/messaging/](../../../docs/messaging/README.md) for user-facing documentation.

## Architecture (current)

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

## Key packages

| Package         | Purpose                                                  |
| --------------- | -------------------------------------------------------- |
| `router/`       | Single owner of publish → NATS → fanout                  |
| `session/`      | Connection registry, subscription cache, egress registry |
| `delivery/`     | `Deliverer` interface implemented by session             |
| `egress/`       | Per-connection outbound pipeline (application layer)     |
| `telemetry/`    | Event types and counter model                            |
| `backpressure/` | Pressure points, drop reasons, rate limit primitives     |
| `envelope/`     | Message envelope with direction and source               |
| `receipt/`      | Ingress ACK/NACK and egress outcomes                     |
| `event/`        | Lifecycle events (not used for message stats)            |

## Statistics

Message stats are **not** updated via the event bus or a stats coordinator. They flow through the telemetry plane:

- Inbound accepted → `msg:in` / `bw:in`
- Outbound delivered → `msg:out` / `bw:out`
- Outbound dropped → `msg:drop`

Gauges (`conn`, `chan`, `sub`) come from repository counts at snapshot time.

See [statistics.md](../../../docs/messaging/statistics.md) and [runtime.md](../../../docs/messaging/runtime.md).

## Contributor guardrails

Do not reintroduce stats via the event bus, stats coordinator, or old Redis keys (`msg:sent`, `msg:rcv`). Do not add routing logic to the WebSocket handler. Full list: [runtime.md](../../../docs/messaging/runtime.md#contributor-guardrails).

## Event bus

Used for lifecycle coordination (channel empty → delayed delete, connection events). Message publish/delivery does not emit stats events.

**Event types:** connection, client, channel, subscription lifecycle only.

## Channel lifecycle

`channel/lifecycle.Manager` listens for `EventChannelEmpty` and deletes channels after the configured grace period (default 30s).

## Related application services

Located under `internal/application/service/messaging/`:

- `router/` — router implementation
- `session/` — session runtime
- `egress/` — outbound pipeline
- `telemetry/` — event subscriber + snapshot
- `backpressure/` — billing plan rate limits

Wiring: `internal/bootstrap/modules/messaging_module.go`
