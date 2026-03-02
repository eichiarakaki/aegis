

```sh
aegis session create <name> --mode <historical/realtime>                              # only creates a session
aegis session create <name> --mode <historical/realtime> --path ./c1 --path ./c2     # creates and runs
aegis session attach <name|id> --path ./c1 --path ./c2                 # attaches components to existent session

aegis session state <name|id>
aegis session start <name|id>
aegis session stop <name|id>
aegis session list
aegis session delete <name|id>

aegis component list <session_name|session_id>
aegis component get <session_name|session_id>
aegis component describe <session_name|session_id>

aegis health check all
aegis health session <name|id>
aegis health component <session_name|session_id> <component>
```
