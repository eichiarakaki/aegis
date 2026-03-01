
## Goal



## Mechanisms

### Sessions


## Responses

**Response structure from the Aegis Daemon to the Aegis CLI:**

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
### Component inspection commands (TO-DO)
COMPONENT_LIST when OK
```json

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