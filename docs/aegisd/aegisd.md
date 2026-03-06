
## Goal



## Mechanisms

### Sessions


## Responses

**Response structure from the Aegis Daemon to the Aegis CTL:**

SESSION_CREATE when OK
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_CREATE",
  "status": "ok",
  "data": {
    "session": {
      "id": "<session_id>",
      "name": "<session_name>",
      "mode": "<session_mode>",
      "state": "initialized",
      "created_at": "<timestamp>"
    }
  }
}
```
SESSION_CREATE when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_CREATE",
  "status": "error",
  "message": "Session creation failed: <reason>",
  "data": {}
}
```

SESSION_ATTACH when OK
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_ATTACH",
  "status": "ok",
  "data": {
    "session_id": "<session_id>",
    "attached_components": [
      { "name": "comp3", "state": "pending" },
      { "name": "comp4", "state": "pending" }
    ]
  }
}
```
SESSION_ATTACH when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_ATTACH",
  "status": "error",
  "message": "Attach failed: <reason>",
  "data": {
    "session_id": "<session_id>"
  }
}
```
SESSION_START when OK
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_START",
  "status": "ok",
  "data": {
    "session_id": "<session_id>",
    "previous_state": "stopped",
    "current_state": "running",
    "started_at": "<timestamp>",
    "components": [
      { "name": "comp1", "state": "running" },
      { "name": "comp2", "state": "running" }
    ]
  }
}
```
SESSION_START when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_START",
  "status": "error",
  "message": "Could not start session: <reason>",
  "data": {
    "session_id": "<session_id>",
    "previous_state": "stopped",
    "current_state": "error",
    "components": [
      { "name": "comp1", "state": "running" },
      { "name": "comp2", "state": "failed" }
    ]
  }
}
```
SESSION_STOP when OK
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_STOP",
  "status": "ok",
  "data": {
    "session_id": "<session_id>",
    "previous_state": "running",
    "current_state": "stopped",
    "stopped_at": "<timestamp>",
    "components": [
      { "name": "comp1", "state": "stopped" },
      { "name": "comp2", "state": "stopped" }
    ]
  }
}
```
SESSION_STOP when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_STOP",
  "status": "error",
  "message": "Could not stop session: <reason>",
  "data": {
    "session_id": "<session_id>"
  }
}
```
SESSION_LIST when OK
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_LIST",
  "status": "ok",
  "data": {
    "sessions": [
      {
        "id": "<session_id>",
        "name": "<session_name>",
        "mode": "<session_mode>",
        "state": "running",
        "created_at": "<timestamp>",
        "started_at": "<timestamp>",
        "stopped_at": null,
        "component_count": 4
      }
    ]
  }
}
```
SESSION_LIST when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_LIST",
  "status": "error",
  "message": "Could not list sessions: <reason>",
  "data": {}
}
```
SESSION_STATE when OK
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_STATE",
  "status": "ok",
  "data": {
    "session": {
      "id": "<session_id>",
      "name": "<session_name>",
      "mode": "<session_mode>",
      "state": "running",
      "uptime_seconds": 384,
      "created_at": "<timestamp>",
      "started_at": "<timestamp>",
      "stopped_at": null
    },
    "components": [
      {
        "name": "comp1",
        "state": "running"
      },
      {
        "name": "comp2",
        "state": "failed"
      }
    ]
  }
}
```
SESSION_STATE when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_STATE",
  "status": "error",
  "message": "Session not found"
}
```
SESSION_DELETE when OK
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_STATE",
  "status": "ok",
  "message": "Session deleted successfully",
  "data": {
    "session_id": "<id>",
    "session_name": "<name>"
  }
}
```
SESSION_STATE when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "SESSION_DELETE",
  "status": "error",
  "message": "Deletion failed: <reason>",
  "data": {}
}
```

### Component inspection commands
COMPONENT_LIST when OK
```json
{
  "request_id": "<uuid>",
  "command": "COMPONENT_LIST",
  "status": "ok",
  "data": {
    "session_id": "<session_id>",
    "components": [
      {
        "id": "<component_id>",
        "name": "data_engine",
        "state": "running",
        "requires": {
          "klines": true,
          "book_depth": true
        },
        "supported_symbols": ["BTCUSDT"],
        "supported_timeframes": ["1m", "15m"]
      },
      {
        "id": "<component_id>",
        "name": "strategy_engine",
        "state": "pending",
        "requires": {
          "klines": true
        },
        "supported_symbols": ["BTCUSDT"],
        "supported_timeframes": ["1m"]
      }
    ]
  }
}
```
COMPONENT_LIST when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "COMPONENT_LIST",
  "status": "error",
  "message": "<message>",
  "data": {}
}
```
COMPONENT_GET when OK
```json
{
  "request_id": "<uuid>",
  "command": "COMPONENT_GET",
  "status": "ok",
  "data": {
    "session_id": "<session_id>",
    "component": {
      "id": "<component_id>",
      "name": "data_engine",
      "state": "running",
      "requires": {
        "klines": true,
        "book_depth": true,
        "trades": false
      },
      "supported_symbols": ["BTCUSDT", "ETHUSDT"],
      "supported_timeframes": ["1m", "5m", "15m"],
      "started_at": "<timestamp>",
      "uptime_seconds": 124
    }
  }
}
```
COMPONENT_GET when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "COMPONENT_GET",
  "status": "error",
  "message": "Component not found in session",
  "data": {
    "session_id": "<session_id>"
  }
}
```
COMPONENT_DESCRIBE when OK
```json
{
  "request_id": "<uuid>",
  "command": "COMPONENT_DESCRIBE",
  "status": "ok",
  "data": {
    "session_id": "<session_id>",
    "component": {
      "id": "<component_id>",
      "name": "data_engine",
      "state": "running",
      "topics_subscribed": [
        "market.BTCUSDT.1m.klines",
        "market.BTCUSDT.orderbook"
      ],
      "topics_published": [
        "session.<session_id>.BTCUSDT.1m.klines"
      ],
      "socket": "/tmp/aegis-data-engine.sock",
      "requires": {
        "klines": true,
        "book_depth": true
      },
      "metrics": {
        "messages_in": 140234,
        "messages_out": 140200,
        "last_heartbeat": "<timestamp>"
      }
    }
  }
}
```
COMPONENT_DESCRIBE when ERROR
```json
{
  "request_id": "<uuid>",
  "command": "COMPONENT_DESCRIBE",
  "status": "error",
  "message": "<message>",
  "data": {}
}
```

## Workflow

0. 

1. El componente se conecta a Aegis

2. El componente envia:
{
    "session_token": "<Gets the session token from the Env>",
    "topics": ["klines.BTCUSDT.1m", "orderbook.BTCUSDT"]
}
2.1 Asigna un UUID al componente como identificador y se inserta a un hashmap de componentes. (Para evitar nombres duplicados)
2.2 El HEARTBEAT comienza simultaneamente

3. Aegis devuelve: {"COMMAND": "ACK"}

4. El componente se conecta al data_socket y configura sus conexiones para los topics

5. El componente se subscribe a los topics

6. El componente se prepara para recibir los datos

6. El componente envia {"COMMAND": "READY"} a Aegis
6.1 Una vez Aegis haya recibido comando: READY, se empezaran a streamear los datos.


7. Aegis empieza a streamear datos a los topics