package core

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
