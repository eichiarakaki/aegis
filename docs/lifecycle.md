### LIFECYCLE FROM START TO FINISH

The user creates and starts a session via the Aegis CLI:
```bash
aegis session create <session_name> --mode <mode>
aegis session attach <session_name|session_id> --path <comp1> --path <comp2>
aegis session start <session_name|session_id>
```

The daemon launches each attached component binary with the following
environment variables injected:

- `AEGIS_SOCKET`        — path to the component manager Unix socket
- `AEGIS_SESSION_TOKEN` — the session ID (used as the registration token)
- `AEGIS_COMPONENT_ID`  — the pre-assigned component ID

---

#### Control channel  (`/tmp/aegis-components.sock`)

All lifecycle, configuration, and heartbeat messages flow over this socket
as newline-delimited JSON envelopes.
```
Component → REGISTER
Aegis     → REGISTERED
Component → STATE_UPDATE(INITIALIZING)
Aegis     → ACK
Component → STATE_UPDATE(READY)
Aegis     → ACK
Aegis     → CONFIGURE
Component → ACK(CONFIGURE)
Component → STATE_UPDATE(CONFIGURED)
Aegis     → ACK
Component → STATE_UPDATE(RUNNING)
Aegis     → ACK
```

**REGISTER** — sent immediately on connect:
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
    "session_token": "<AEGIS_SESSION_TOKEN>",
    "component_id":  "<AEGIS_COMPONENT_ID>",
    "component_name": "data_engine",
    "version": "0.1.0",
    "capabilities": {
      "supported_symbols":    ["BTCUSDT", "ETHUSDT"],
      "supported_timeframes": ["1m", "15m", "1h"],
      "requires_streams":     ["klines", "orderbook"]
    }
  }
}
```

**REGISTERED** — Aegis confirms registration:
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
    "session_id":   "sess-xyz",
    "state":        "REGISTERED"
  }
}
```

**STATE_UPDATE(INITIALIZING) / STATE_UPDATE(READY)** — the component signals
it is initializing its internal resources, then ready to receive configuration.
Aegis ACKs each one:
```json
{
  "type": "LIFECYCLE",
  "command": "STATE_UPDATE",
  "payload": { "state": "INITIALIZING" }
}
```

**CONFIGURE** — Aegis sends the data stream socket path and the topic list
assigned to this component:
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
    "data_stream_socket": "/tmp/aegis-data-stream-sess-xyz.sock",
    "topics": ["klines.BTCUSDT.1m", "orderbook.BTCUSDT"]
  }
}
```

The component ACKs CONFIGURE, then sends STATE_UPDATE(CONFIGURED) and
STATE_UPDATE(RUNNING). Aegis ACKs each state update.

---

#### Heartbeat

While connected, Aegis sends a PING every 5 seconds. The component must
reply with a PONG within 15 seconds or it will be considered dead and
unregistered:
```
Aegis     → PING
Component → PONG
```
```json
{
  "type": "HEARTBEAT",
  "command": "PING",
  "payload": {}
}
```
```json
{
  "type": "HEARTBEAT",
  "command": "PONG",
  "payload": {
    "state":          "RUNNING",
    "uptime_seconds": 123
  }
}
```

---

#### Data stream  (`/tmp/aegis-data-stream-<session_id>.sock`)

A separate Unix socket used exclusively for market data. The component
connects to it **after** receiving CONFIGURE and performs a one-time
handshake before data starts flowing:
```
Component → {"component_id": "cmp-abc123", "session_token": "sess-xyz"}
Server    → {"status": "ok", "topics": ["klines.BTCUSDT.1m", ...]}
```

After the handshake the server streams newline-delimited JSON frames:
```json
{"topic": "klines.BTCUSDT.1m", "data": { ... }}
```

The stream is unidirectional — the component only reads after the handshake.
If the socket closes, the component reconnects and re-handshakes.

---

#### Session restart  (REBORN)

When a finished session is restarted (`aegis session start` on a FINISHED
session), Aegis sends REBORN instead of going through the full lifecycle
again. The component clears its internal state and ACKs — no reconnect or
new CONFIGURE is needed. The data stream socket is recreated by the server
and the component reconnects to it automatically.
```
Aegis     → REBORN
Component → ACK
```

---

#### Error reporting

The component can report runtime errors at any time:
```json
{
  "type": "ERROR",
  "command": "RUNTIME_ERROR",
  "payload": {
    "code":        "STREAM_CONNECTION_FAILED",
    "message":     "Failed to connect to stream socket",
    "recoverable": true
  }
}
```

Non-recoverable errors cause Aegis to shut the component down.