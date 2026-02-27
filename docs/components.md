### Goal
El objetivo de conectar los componentes al socket /tmp/aegis-components.sock es registarse en Aegis y mantener una comunicación abierta para recibir o enviar COMANDOS.
Ejemplo de comandos del componente a Aegis:
- {"COMMAND": "READY"}
- {"COMMAND": "ACK"}
- {"COMMAND": "WAIT"}
- {"COMMAND": "ERROR"}
- {"COMMAND": "FINISHED"}
Ejemplo de comandos de Aegis al componente:
- {"COMMAND": "HEARTBEAT"}
- {"COMMAND": "STARTED"}
- {"COMMAND": "WAIT"}
- {
    "data_socket": "/tmp/aegis-data-stream-<id>.sock",
    "topics": ["klines.BTCUSDT.1m", "orderbook.BTCUSDT"]
}

### Workflow

Ejemplo de flujo:
1. El componente se conecta a Aegis
2. El componente envia:
{
    "data_socket": "/tmp/aegis-data-<id>.sock",
    "topics": ["klines.BTCUSDT.1m", "orderbook.BTCUSDT"]
}
3. Aegis revuelve: {"COMMAND": "ACK"}
4. El componente se conecta al data_socket y configura sus conexiones para los topics
5. El componente se subscribe a los topics
6. El componente se prepara para recibir los datos
6. El componente envia {"COMMAND": "READY"} a Aegis
7. Aegis empieza a streamear datos a los topics


---

```yaml
components_socket: "/tmp/aegis-components.sock" # This is the socket where all components will register themselves to the communication engine. The communication engine will keep track of all registered components and their capabilities (e.g., which symbols and timeframes they support).
# Example of expected payload: 
# {
#   component_name: "data_engine", 
#   supported_symbols: ["BTCUSDT", "ETHUSDT"], 
#   requires: ["klines", "orderbook"], 
#   supported_timeframes: ["1m", "15m", "30m", "4h", "1d"]
# }
```
