

```sh
aegis session create <name> --mode live                              # only creates a session
aegis session create <name> --mode live --path ./c1 --path ./c2     # creates and runs
aegis session attach <name> --path ./c1 --path ./c2                 # attaches components to existent session
aegis session status <name|id>
aegis session start <id>
aegis session stop <id>
aegis session list
aegis session delete <id>

aegis health check <all|data|sesions>
```