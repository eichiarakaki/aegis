
## Goal



## Mechanisms

### Sessions




## Workflow

0. 

1. El componente se conecta a Aegis

2. El componente envia:
{
    "data_socket": "/tmp/aegis-data-<id>.sock",
    "topics": ["klines.BTCUSDT.1m", "orderbook.BTCUSDT"]
}
2.1 Asigna un UUID al componente como identificador y se inserta a un hashmap de componentes. (Para evitar nombres duplicados)
2.2 El HEARTBEAT comienza simultaneamente

3. Aegis revuelve: {"COMMAND": "ACK"}

4. El componente se conecta al data_socket y configura sus conexiones para los topics

5. El componente se subscribe a los topics

6. El componente se prepara para recibir los datos

6. El componente envia {"COMMAND": "READY"} a Aegis
6.1 Una vez Aegis haya recibido comando: READY, se empezaran a streamear los datos.


7. Aegis empieza a streamear datos a los topics