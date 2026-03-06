

```sh
aegisctl session create <name> --mode <historical/realtime>                              # only creates a session
# aegisctl session create <name> --mode <historical/realtime> --path ./c1 --path ./c2     # creates and runs
aegisctl session attach <name|id> --path ./c1 --path ./c2                 # attaches components to existent session

aegisctl session state <name|id>
aegisctl session start <name|id>
aegisctl session stop <name|id>
aegisctl session list
aegisctl session delete <name|id>

aegisctl component list <session_name|session_id>
aegisctl component get <session_name|session_id>
aegisctl component describe <session_name|session_id>

aegisctl health check all
aegisctl health session <name|id>
aegisctl health component <session_name|session_id> <component>
```
