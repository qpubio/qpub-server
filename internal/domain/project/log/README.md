# Project Log Domain

This domain handles logging and broadcasting of project-related events, particularly connection lifecycle events.

## Overview

The project log domain provides functionality to:
- Define log event types for connection lifecycle events
- Broadcast log events to the `_logs` channel in real-time
- Track connection, client, and subscription lifecycle events

## Components

### Event Model (`model.go`)

Defines the structure for log events and event types.

**Event Types:**
- `connection.opened` - WebSocket connection established
- `connection.closed` - Connection closed gracefully
- `connection.error` - Error occurred on connection
- `client.connected` - Client authenticated and connected
- `client.disconnected` - Client disconnected
- `subscription.created` - Client subscribed to a channel
- `subscription.closed` - Client unsubscribed from a channel

**Event Structure:**
```go
type Event struct {
    ProjectID    id.Int                 // Project ID
    EventType    EventType              // Type of event
    Timestamp    time.Time              // When event occurred
    ConnectionID *id.ULID               // Optional connection ID
    ClientID     *id.ULID               // Optional client ID
    ChannelName  *string                // Optional channel name
    Message      string                 // Human-readable message
    Metadata     map[string]interface{} // Additional event data
}
```

### Broadcast Service (`broadcast/service.go`)

Provides the interface and implementation for broadcasting log events to the `_logs` channel.

**Service Interface:**
```go
type Service interface {
    PublishLog(event log.Event) error
}
```

## Usage

### Basic Example

```go
// Create log broadcaster (usually injected via DI)
logBroadcaster := modules.NewProjectLogBroadcastModule(logger, publicationService)

// Create and publish a connection opened event
event := log.CreateEvent(log.CreateEventParams{
    ProjectID:    projectID,
    EventType:    log.EventConnectionOpened,
    ConnectionID: &connectionID,
    Message:      "Connection established",
    Metadata: map[string]interface{}{
        "remote_addr": "192.168.1.1",
        "user_agent":  "Mozilla/5.0...",
    },
})

if err := logBroadcaster.PublishLog(*event); err != nil {
    logger.Warn(log.ProjectLog, "Failed to broadcast event", "error", err)
}
```

### Integration Points

The log broadcast service is typically integrated into:

1. **Connection Service** - Track connection open/close events
2. **Client Service** - Track client connect/disconnect events
3. **Subscription Service** - Track subscribe/unsubscribe events
4. **WebSocket Handler** - Track error events

### Bootstrap Setup

To use the log broadcast service, wire it up in your bootstrap:

```go
// In bootstrap/tasks.go or similar
publicationService, err := container.GetTyped[publication.Service](a.container)
if err != nil {
    return fmt.Errorf("failed to get publication service: %w", err)
}

// Create log broadcast service
logBroadcaster := modules.NewProjectLogBroadcastModule(a.logger, publicationService)

// Inject into services that need it
connectionService := connectionService.NewService(
    logger,
    repository,
    statService,
    instanceID,
    logBroadcaster, // Add as dependency
)
```

## Channel Details

**Channel Name:** `_logs`

**Event Format:**
```json
{
    "event": "connection.opened",
    "data": {
        "project_id": 123,
        "event": "connection.opened",
        "timestamp": "2025-10-08T12:34:56Z",
        "connection_id": "01H1234567890ABCDEF",
        "message": "Connection established",
        "metadata": {
            "remote_addr": "192.168.1.1"
        }
    }
}
```

## Best Practices

1. **Non-blocking** - Broadcasting should not block the main operation. Log failures, don't return errors.
2. **Minimal data** - Include only essential information in metadata to keep messages small.
3. **Consistent naming** - Use the predefined EventType constants.
4. **Graceful degradation** - If broadcast fails, the primary operation should continue.
5. **Stats exclusion** - Log broadcasts use `skipStats: true` to avoid affecting project statistics.

## Testing

When testing services that use log broadcasting:

1. Mock the broadcast service
2. Verify events are created with correct data
3. Ensure failures don't break the main flow

```go
mockBroadcaster := &MockLogBroadcaster{}
mockBroadcaster.On("PublishLog", mock.Anything).Return(nil)

// Use in tests
service := NewService(logger, repository, mockBroadcaster)

// Verify broadcast was called
mockBroadcaster.AssertCalled(t, "PublishLog", mock.MatchedBy(func(e log.Event) bool {
    return e.EventType == log.EventConnectionOpened
}))
```

## Related Domains

- `project/usage/broadcast` - Similar pattern for usage data broadcasting
- `messaging/connection` - Connection lifecycle management
- `messaging/client` - Client connection management
- `messaging/subscription` - Subscription management

