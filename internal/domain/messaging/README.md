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

Message stats flow through the telemetry plane (not the event bus):

- Inbound accepted → `msg:in` / `bw:in`
- Outbound delivered → `msg:out` / `bw:out`
- Outbound dropped → `msg:drop`

Gauges (`conn`, `chan`, `sub`) come from repository counts at snapshot time.

## Related application services

Under `internal/application/service/messaging/`:

- `router/` — router implementation
- `session/` — session runtime
- `egress/` — outbound pipeline
- `telemetry/` — event subscriber + snapshot
- `backpressure/` — tenant rate limits

Wiring: `internal/bootstrap/modules/messaging_module.go`
