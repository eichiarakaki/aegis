# WorkFlow


---

## Technical criteria

### Backtesting Mode - Topics synchronization

Todos los topics deben estar 100% sincronizados, dependiendo del tamaño de los datos, un topic puede terminar de mandar todos datos mas rapido que otro al Strategy Engine, pero eso es un grave error si hay topics para más de un tipo de dato, por ejemplo un topic de klines y un topic de orderbook, tendran diferentes velocidades de envio, de nada le servidiria al strategy engine que le llegue un kline del año 2020 y a la vez un order del año 2022.
