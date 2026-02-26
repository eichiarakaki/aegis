# WorkFlow

1. Program starts

2. Program initiates Live mode and Backtest mode

---

## Live mode

1. Reads `live` and `currencies` from the mercury.yaml configuration file
2. Reads `components_socket` and `mercury_cli_socket` from the globals.yaml configuration file.
3. (Health Check) Verify that all currencies in mercury.yaml are available at the broker.
4. Dispatch
    1. Connects to the broker and stays in a loop
    2. waits for component connections
    3. Waits to get the Start from the trigger
        1. Gets a payload from the Trigger specifying symbols, timeframes, etc.
    4. Dispatch the data

---

## Backtesting Mode

1. Reads `backtesting` and `currencies` from the mercury.yaml configuration file.
2.
3. Reads from `start_date` and `end_date`

4. If the data is too big, it will be split into chunks and load only the required amount.

---

## Technical criteria

### Backtesting Mode - Topics synchronization

Todos los topics deben estar 100% sincronizados, dependiendo del tamaño de los datos, un topic puede terminar de mandar todos datos mas rapido que otro al Strategy Engine, pero eso es un grave error si hay topics para más de un tipo de dato, por ejemplo un topic de klines y un topic de orderbook, tendran diferentes velocidades de envio, de nada le servidiria al strategy engine que le llegue un kline del año 2020 y a la vez un order del año 2022.
