package component

import (
	"time"

	"github.com/eichiarakaki/aegis/internals/services/utils"
)

/* STANDARD ENVELOPE
{
  "protocol_version": "1.0",
  "message_id": "uuid",
  "correlation_id": "uuid | null",
  "timestamp": "2026-02-27T12:00:00Z",
  "source": "component:data_engine",
  "target": "aegis",
  "type": "CONTROL | LIFECYCLE | CONFIG | ERROR | HEARTBEAT",
  "command": "COMMAND_NAME",
  "payload": {}
}
*/

const ProtocolVersion = "0.1.0"

// MessageType classifies the message type in the protocol
type MessageType string

const (
	MessageTypeControl   MessageType = "CONTROL"
	MessageTypeLifecycle MessageType = "LIFECYCLE"
	MessageTypeConfig    MessageType = "CONFIG"
	MessageTypeError     MessageType = "ERROR"
	MessageTypeHeartbeat MessageType = "HEARTBEAT"
	MessageTypeData      MessageType = "DATA"
)

// ComponentState represents the possible states of a component
type ComponentState string

const (
	ComponentStateInit         ComponentState = "INIT"
	ComponentStateRegistered   ComponentState = "REGISTERED"
	ComponentStateInitializing ComponentState = "INITIALIZING"
	ComponentStateReady        ComponentState = "READY"
	ComponentStateConfigured   ComponentState = "CONFIGURED"
	ComponentStateRunning      ComponentState = "RUNNING"
	ComponentStateWaiting      ComponentState = "WAITING"
	ComponentStateError        ComponentState = "ERROR"
	ComponentStateFinished     ComponentState = "FINISHED"
	ComponentStateShutdown     ComponentState = "SHUTDOWN"
)

// CommandType represents the specific commands of the protocol
type CommandType string

const (
	// Lifecycle commands
	CommandRegister    CommandType = "REGISTER"
	CommandRegistered  CommandType = "REGISTERED"
	CommandStateUpdate CommandType = "STATE_UPDATE"
	CommandShutdown    CommandType = "SHUTDOWN"

	// Control commands
	CommandACK  CommandType = "ACK"
	CommandNACK CommandType = "NACK"

	// Config commands
	CommandConfigure  CommandType = "CONFIGURE"
	CommandConfigured CommandType = "CONFIGURED"

	// Heartbeat commands
	CommandPing CommandType = "PING"
	CommandPong CommandType = "PONG"

	// Error commands
	CommandRuntimeError       CommandType = "RUNTIME_ERROR"
	CommandRegistrationFailed CommandType = "REGISTRATION_FAILED"
)

// Envelope is the standard structure for ALL the messages
type Envelope struct {
	ProtocolVersion string         `json:"protocol_version"`
	MessageID       string         `json:"message_id"`
	CorrelationID   *string        `json:"correlation_id"`
	Timestamp       string         `json:"timestamp"`
	Source          string         `json:"source"`
	Target          string         `json:"target"`
	Type            MessageType    `json:"type"`
	Command         CommandType    `json:"command"`
	Payload         map[string]any `json:"payload"`
}

// RegisterPayload is the payload for the REGISTER command
type RegisterPayload struct {
	SessionToken  string                `json:"session_token"`
	ComponentName string                `json:"component_name"`
	ComponentID   string                `json:"component_id,omitempty"`
	Version       string                `json:"version"`
	Capabilities  ComponentCapabilities `json:"capabilities"`
}

// ComponentCapabilities describes what the component can do
type ComponentCapabilities struct {
	SupportedSymbols    []string `json:"supported_symbols"`
	SupportedTimeframes []string `json:"supported_timeframes"`
	RequiresStreams     []string `json:"requires_streams"`
}

// RegisteredPayload is the REGISTER response
type RegisteredPayload struct {
	ComponentID string         `json:"component_id"`
	SessionID   string         `json:"session_id"`
	State       ComponentState `json:"state"`
}

// StateUpdatePayload to notify state change
type StateUpdatePayload struct {
	State         ComponentState `json:"state"`
	UptimeSeconds *int64         `json:"uptime_seconds,omitempty"`
	Message       *string        `json:"message,omitempty"`
}

// ConfigurePayload is the configuration sent by Aegis
type ConfigurePayload struct {
	DataStreamSocket string   `json:"data_stream_socket"`
	Topics           []string `json:"topics"`
}

// ACKPayload is the confirmation of a command
type ACKPayload struct {
	Status  string  `json:"status"`
	Message *string `json:"message,omitempty"`
}

// PongPayload is the response of PING
type PongPayload struct {
	State         ComponentState `json:"state"`
	UptimeSeconds int64          `json:"uptime_seconds"`
}

// ErrorPayload contains detailed information of an error
type ErrorPayload struct {
	Code        string                  `json:"code"`
	Message     string                  `json:"message"`
	Recoverable bool                    `json:"recoverable"`
	Details     *map[string]interface{} `json:"details,omitempty"`
}

// NewEnvelope creates a new envelope with default values
func NewEnvelope(
	messageType MessageType,
	command CommandType,
	source string,
	target string,
	payload map[string]interface{},
) *Envelope {
	return &Envelope{
		ProtocolVersion: "0.1",
		MessageID:       utils.GenerateSecureToken(),
		CorrelationID:   nil,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Source:          source,
		Target:          target,
		Type:            messageType,
		Command:         command,
		Payload:         payload,
	}
}

// WithCorrelation add a correlation_id (for responses)
func (e *Envelope) WithCorrelation(correlationID string) *Envelope {
	e.CorrelationID = &correlationID
	return e
}

// Validate validates an envelope
func (e *Envelope) Validate() error {
	if e.ProtocolVersion == "" {
		return NewValidationError("MISSING_PROTOCOL_VERSION", "protocol_version is required")
	}

	if e.MessageID == "" {
		return NewValidationError("MISSING_MESSAGE_ID", "message_id is required")
	}

	if e.Source == "" {
		return NewValidationError("MISSING_SOURCE", "source is required")
	}

	if e.Target == "" {
		return NewValidationError("MISSING_TARGET", "target is required")
	}

	if e.Type == "" {
		return NewValidationError("MISSING_TYPE", "type is required")
	}

	if e.Command == "" {
		return NewValidationError("MISSING_COMMAND", "command is required")
	}

	if e.Payload == nil {
		return NewValidationError("MISSING_PAYLOAD", "payload is required")
	}

	return nil
}

// ValidationError is an error of a specific validation
type ValidationError struct {
	Code    string
	Message string
}

func (v ValidationError) Error() string {
	return v.Code + ": " + v.Message
}

func NewValidationError(code, message string) ValidationError {
	return ValidationError{
		Code:    code,
		Message: message,
	}
}
