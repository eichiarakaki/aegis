package core

// CommandType holds the string identifier for a CLI/daemon command.
// Keeping them as constants avoids typos across cmd, server and tests.
type CommandType string

const (
	// Daemon lifecycle
	CommandDaemonShutdown CommandType = "DAEMON_SHUTDOWN"
	CommandDaemonKill     CommandType = "DAEMON_KILL"

	// Sessions
	CommandSessionCreate CommandType = "SESSION_CREATE"
	CommandSessionAttach CommandType = "SESSION_ATTACH"
	CommandSessionStart  CommandType = "SESSION_START"
	CommandSessionStop   CommandType = "SESSION_STOP"
	CommandSessionList   CommandType = "SESSION_LIST"
	CommandSessionState  CommandType = "SESSION_STATE"
	CommandSessionDelete CommandType = "SESSION_DELETE"

	// Components
	CommandComponentList      CommandType = "COMPONENT_LIST"
	CommandComponentGet       CommandType = "COMPONENT_GET"
	CommandComponentDescribe  CommandType = "COMPONENT_DESCRIBE"
	CommandComponentLogs      CommandType = "COMPONENT_LOGS"
	CommandComponentLogPath   CommandType = "COMPONENT_LOG_PATH"
	CommandHealthCheck        CommandType = "HEALTH_CHECK"
	CommandHealthCheckSession CommandType = "HEALTH_CHECK_SESSION"
	CommandHealthCheckComp    CommandType = "HEALTH_CHECK_COMPONENT"
)

type Command struct {
	RequestID string      `json:"request_id"`
	Type      CommandType `json:"type"`
	Payload   interface{} `json:"payload"`
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
