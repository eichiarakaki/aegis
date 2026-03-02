### LIFECYCLE FROM START TO FINISH

User gives a command via the aegis CLI:
```bash
aegis session create <session_name> --mode <mode>
aegis session attach <session_name|session_id> --path <comp1> -- <comp2>
```

> Then the Aegis Daemon generates a SessionToken and runs the components with an Env variable with the same SessionToken.

> When the component is run, it connects to the /tmp/aegis-components.sock and sends a JSON:
```json
{
  "protocol_version": "0.1.0",
  "message_id": "uuid-1",
  "correlation_id": null,
  "timestamp": "2026-02-27T12:00:00Z",
  "source": "component:data_engine",
  "target": "aegis",
  "type": "LIFECYCLE",
  "command": "REGISTER",
  "payload": {
    "session_token": "<env_session_token>",
    "component_name": "data_engine",
    "capabilities": {
      "supported_symbols": ["BTCUSDT", "ETHUSDT"],
      "supported_timeframes": ["1m", "15m", "1h"],
      "requires_streams": ["klines", "orderbook"]
    },
    "version": "0.1.0"
  }
}
```

> If an error occurs while verifying the session token
```json
{
  "type": "ERROR",
  "command": "REGISTRATION_FAILED",
  "payload": {
    "reason": "INVALID_SESSION_TOKEN"
  }
}
```

> When the SessionToken of the Aegis Daemon and the one that the components has matches:

**Aegis Daemon do**
1. Creates a core.Component and fills it with the component's data.
2. Adds it to the respective session.
3. Returns a JSON with the following structure to the client:
```json
{
  "protocol_version": "0.1.0",
  "message_id": "uuid-2",
  "correlation_id": "uuid-1",
  "timestamp": "2026-02-27T12:00:01Z",
  "source": "aegis",
  "target": "component:data_engine",
  "type": "LIFECYCLE",
  "command": "REGISTERED",
  "payload": {
    "component_id": "cmp-abc123",
    "session_id": "sess-xyz",
    "state": "REGISTERED"
  }
}
```

> Then, when the component gets ready, it sends the following JSON structure to Aegis
```json
{
  "protocol_version": "0.1.0",
  "message_id": "uuid-3",
  "correlation_id": null,
  "timestamp": "2026-02-27T12:00:02Z",
  "source": "component:data_engine",
  "target": "aegis",
  "type": "LIFECYCLE",
  "command": "STATE_UPDATE",
  "payload": {
    "state": "READY"
  }
}
```

> Then, when Aegis acknowledge that, it sends the following JSON structure to the component
```json
{
  "protocol_version": "0.1.0",
  "message_id": "uuid-4",
  "correlation_id": null,
  "timestamp": "2026-02-27T12:00:03Z",
  "source": "aegis",
  "target": "component:data_engine",
  "type": "CONFIG",
  "command": "CONFIGURE",
  "payload": {
    "data_stream_socket": "/tmp/aegis-data-stream-123.sock",
    "topics": [
      "klines.BTCUSDT.1m",
      "orderbook.BTCUSDT"
    ]
  }
}
```

> Then, the component responds with the following JSON structure to Aegis
```json
{
  "protocol_version": "0.1.0",
  "message_id": "uuid-5",
  "correlation_id": "uuid-4",
  "timestamp": "2026-02-27T12:00:04Z",
  "source": "component:data_engine",
  "target": "aegis",
  "type": "CONTROL",
  "command": "ACK",
  "payload": {
    "status": "ok"
  }
}
```
 
---

> While the component is connected to Aegis, Aegis will be sending heartbeats to the component with the following JSON structure
```json
{
  "protocol_version": "1.0",
  "message_id": "uuid-6",
  "correlation_id": null,
  "timestamp": "...",
  "source": "aegis",
  "target": "component:data_engine",
  "type": "HEARTBEAT",
  "command": "PING",
  "payload": {}
}
```

> The component has to respond with
```json
{
  "protocol_version": "1.0",
  "message_id": "uuid-7",
  "correlation_id": "uuid-6",
  "timestamp": "...",
  "source": "component:data_engine",
  "target": "aegis",
  "type": "HEARTBEAT",
  "command": "PONG",
  "payload": {
    "state": "RUNNING",
    "uptime_seconds": 123
  }
}
```

> In case of errors from the component
```json
{
  "protocol_version": "1.0",
  "message_id": "uuid-8",
  "correlation_id": "uuid-4",
  "timestamp": "...",
  "source": "component:data_engine",
  "target": "aegis",
  "type": "ERROR",
  "command": "RUNTIME_ERROR",
  "payload": {
    "code": "STREAM_CONNECTION_FAILED",
    "message": "Failed to connect to stream socket",
    "recoverable": true
  }
}
```