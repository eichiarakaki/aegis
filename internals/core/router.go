package core

import (
	"encoding/json"
	"fmt"
)

type Command struct {
	RequestID string         `json:"request_id"`
	Type      CLICommandType `json:"type"`
	Payload   interface{}    `json:"payload"`
}

// Specific payload for each command

type SessionCreatePayload struct {
	Name string `json:"name"`
	Mode string `json:"mode"`
}

type SessionCreateRunPayload struct {
	Name  string   `json:"name"`
	Mode  string   `json:"mode"`
	Paths []string `json:"paths"`
}

type SessionAttachPayload struct {
	SessionID string   `json:"session_id"`
	Paths     []string `json:"paths"`
}

type SessionActionPayload struct {
	SessionID string `json:"session_id"`
}

type ComponentListPayload struct {
	SessionID string `json:"session_id"`
}

type ComponentGetPayload struct {
	SessionID   string `json:"session_id"`
	ComponentID string `json:"component_id"`
}

type HealthCheckPayload struct {
	Target string `json:"target"`
}

type HealthCheckSessionPayload struct {
	SessionID string `json:"session_id"`
}

type HealthCheckComponentPayload struct {
	SessionID   string `json:"session_id"`
	ComponentID string `json:"component_id"`
}

// Useful functions

func DeserializeSessionActionPayload(cmd Command) (*SessionActionPayload, error) {
	var payload SessionActionPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload %s", err.Error())
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("payload parsing error: %s", err.Error())
	}

	// Validate required field
	if payload.SessionID == "" {
		return nil, fmt.Errorf("missing required field: session_id")
	}

	return &payload, nil
}

func DeserializeSessionAttachPayload(cmd Command) (*SessionAttachPayload, error) {
	// Deserialize payload
	var payload SessionAttachPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		return nil, fmt.Errorf("payload parsing error: %s", err.Error())
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("payload parsing error: %s", err.Error())
	}

	// Validate required fields
	if payload.SessionID == "" {
		return nil, fmt.Errorf("missing required field: session_id")
	}

	return &payload, nil
}

func DeserializeSessionStartPayload(cmd Command) (*SessionStartPayload, error) {
	var payload SessionStartPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		return nil, fmt.Errorf("invalid payload format")
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("payload parsing error: %s", err.Error())
	}

	if payload.SessionID == "" {
		return nil, fmt.Errorf("missing required field: session_id")
	}

	return &payload, nil
}
