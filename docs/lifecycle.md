### LIFECYCLE FROM START TO FINISH

User gives a command via the aegis CLI:
```bash
aegis session create <session_name> --mode <mode> --path <path/to/comp1> --path <path/to/comp2>
```
or
```bash
aegis session attach <existing_session_name or session_id> --path <path/to/comp1> --path <path/to/comp2>
```

Then the Aegis Daemon generates a SessionToken and runs the components with an Env variable with the same SessionToken, and when the SessionToken of the Aegis Daemon and the one that the components sends match: They connect to the /tmp/aegis-data-stream-<session_id>.sock

NOTE: All the components needs to have something in common: Same Environment variable handling (to send to the /tmp/aegis-components.sock) and proper socket communication (specified in docs/components.md)