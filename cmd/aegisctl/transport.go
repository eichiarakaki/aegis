package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/google/uuid"
)

// requestJSON sends a CLI command over the Unix socket and returns the decoded
// response. A nil response map (with no error) means the daemon sent no body.
func requestJSON(cmdType core.CLICommandType, payload interface{}) (map[string]any, error) {
	cfg, err := config.LoadGlobals()
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial("unix", cfg.AegisCLISocket)
	if err != nil {
		return nil, fmt.Errorf("connect to daemon socket: %w", err)
	}
	defer conn.Close()

	cmd := core.Command{
		RequestID: uuid.NewString(),
		Type:      cmdType,
		Payload:   payload,
	}

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		return nil, fmt.Errorf("encode command: %w", err)
	}

	var response map[string]any
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		// Empty body is not an error — daemon may send nothing on success.
		return nil, nil
	}
	return response, nil
}

// sendCommand is a convenience wrapper around requestJSON that renders the
// response using the human-friendly prettyPrint formatter.
func sendCommand(cmdType core.CLICommandType, payload interface{}) error {
	resp, err := requestJSON(cmdType, payload)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	prettyPrint(resp)
	return nil
}
