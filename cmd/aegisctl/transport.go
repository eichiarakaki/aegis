package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/google/uuid"
)

func requestJSON(cmdType core.CLICommandType, payload interface{}) (map[string]any, error) {
	cfg, err := config.LoadGlobals()
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial("unix", cfg.AegisCTLSocket)
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
		return nil, nil
	}
	return response, nil
}

func sendCommand(cmdType core.CLICommandType, payload interface{}) error {
	resp, err := requestJSON(cmdType, payload)
	if err != nil {
		return err
	}
	if resp == nil {
		fmt.Fprintln(os.Stderr, "[debug] daemon returned no response body")
		return nil
	}

	if core.DebugEnabled {
		raw, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Fprintln(os.Stderr, string(raw))
	}
	prettyPrint(resp)
	return nil
}
