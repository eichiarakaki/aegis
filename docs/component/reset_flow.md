Restart flow:
When a session finishes (all data exhausted), it transitions to FINISHED. A restart does not tear down component processes or connections - it only resets orchestration and component state.

The old orchestrator and data stream server are stopped
Session state resets to INITIALIZED
Aegis sends REBORN to each registered component over the existing TCP connection and waits for ACK
On the component side, on_reborn() is called - the handler clears all per-run domain state (positions, POC levels, absorption buffers, etc.). The on_running task is not interrupted; it keeps blocking on the stream socket
A new data stream server and orchestrator are created with the new TimeRange
Session transitions to RUNNING
The new orchestrator begins publishing data - the component receives it on the same socket it has been connected to since initial startup, with no reconnection or re-handshake