
```json
{
  "protocol_version": "1.0",
  "message_id": "uuid",
  "correlation_id": "uuid | null",
  "timestamp": "2026-02-27T12:00:00Z",
  "source": "component:data_engine",
  "target": "aegis",
  "type": "CONTROL | LIFECYCLE | CONFIG | ERROR | HEARTBEAT",
  "command": "COMMAND_NAME",
  "payload": {}
}
```

* `type` clasifica la naturaleza del mensaje.
* `command` define la acción específica.
* `payload` contiene los datos concretos.

---

# `type`: LIFECYCLE

## Para qué sirve

Define cambios de estado del componente o sesión.

Es la capa de "vida y muerte" del sistema.

Sirve para:

* Registro
* Cambios de estado
* Inicio / fin
* Shutdown
* Declarar que estás listo

---

## Ejemplo: REGISTER

```json
{
  "type": "LIFECYCLE",
  "command": "REGISTER",
  "payload": {
    "session_token": "abc123",
    "component_name": "data_engine",
    "version": "0.1.0"
  }
}
```

---

## Ejemplo: STATE_UPDATE

```json
{
  "type": "LIFECYCLE",
  "command": "STATE_UPDATE",
  "payload": {
    "state": "READY"
  }
}
```

---

## Ejemplo: FINISHED

```json
{
  "type": "LIFECYCLE",
  "command": "FINISHED",
  "payload": {
    "reason": "Backtest completed"
  }
}
```

---

# `type`: CONTROL

## Para qué sirve

Controla comportamiento puntual.

No cambia estado global.
No es configuración.
No es error.
No es heartbeat.

Es comando operativo inmediato.

---

## Ejemplo: ACK

Responder a una orden.

```json
{
  "type": "CONTROL",
  "command": "ACK",
  "payload": {
    "status": "ok"
  }
}
```

---

## Ejemplo: PAUSE

```json
{
  "type": "CONTROL",
  "command": "PAUSE",
  "payload": {}
}
```

---

## Ejemplo: RESUME

```json
{
  "type": "CONTROL",
  "command": "RESUME",
  "payload": {}
}
```

---

# `type`: CONFIG

## 📌 Para qué sirve

Envía configuración estructural.

Esto define cómo debe funcionar el componente.

Se usa cuando:

* Se asignan sockets
* Se asignan topics
* Se envían parámetros
* Se modifica comportamiento

---

## Ejemplo: CONFIGURE

```json
{
  "type": "CONFIG",
  "command": "CONFIGURE",
  "payload": {
    "data_stream_socket": "/tmp/aegis-stream.sock",
    "topics": ["klines.BTCUSDT.1m"],
    "buffer_size": 10000
  }
}
```

---

## Ejemplo: UPDATE_CONFIG

```json
{
  "type": "CONFIG",
  "command": "UPDATE",
  "payload": {
    "symbols": ["BTCUSDT", "ETHUSDT"]
  }
}
```

---

# `type`: ERROR

## Para qué sirve

Errores estructurados.

No es un log.
No es un print.
Es parte del protocolo.

Sirve para:

* Fallos de registro
* Fallos de ejecución
* Validaciones
* Errores recuperables o fatales

---

## Ejemplo: REGISTRATION_FAILED

```json
{
  "type": "ERROR",
  "command": "REGISTRATION_FAILED",
  "payload": {
    "code": "INVALID_SESSION_TOKEN",
    "message": "Session token is not valid",
    "recoverable": false
  }
}
```

---

## Ejemplo: RUNTIME_ERROR

```json
{
  "type": "ERROR",
  "command": "RUNTIME_ERROR",
  "payload": {
    "code": "STREAM_DISCONNECTED",
    "message": "Lost connection to data stream",
    "recoverable": true
  }
}
```

---

# `type`: HEARTBEAT

## Para qué sirve

Mantener viva la conexión.

Detectar componentes muertos.
Medir latencia.
Monitorear salud.

No cambia estado.
No ejecuta lógica.
Solo monitorea.

---

## Ejemplo: PING

```json
{
  "type": "HEARTBEAT",
  "command": "PING",
  "payload": {}
}
```

---

## Ejemplo: PONG

```json
{
  "type": "HEARTBEAT",
  "command": "PONG",
  "payload": {
    "state": "RUNNING",
    "uptime_seconds": 452
  }
}
```

---

```
switch msg.Type {
case LIFECYCLE:
    handleLifecycle(msg)
case CONFIG:
    handleConfig(msg)
case CONTROL:
    handleControl(msg)
case ERROR:
    handleError(msg)
case HEARTBEAT:
    handleHeartbeat(msg)
}
```
