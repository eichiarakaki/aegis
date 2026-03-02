package component

// EnvelopeBuilder facilita la construcción de envelopes
type EnvelopeBuilder struct {
	envelope *Envelope
}

// NewBuilder crea un nuevo builder
func NewBuilder(messageType MessageType, command CommandType, source, target string) *EnvelopeBuilder {
	return &EnvelopeBuilder{
		envelope: NewEnvelope(messageType, command, source, target, make(map[string]interface{})),
	}
}

// WithPayload establece el payload
func (b *EnvelopeBuilder) WithPayload(payload map[string]interface{}) *EnvelopeBuilder {
	b.envelope.Payload = payload
	return b
}

// WithCorrelation establece el correlation_id
func (b *EnvelopeBuilder) WithCorrelation(correlationID string) *EnvelopeBuilder {
	b.envelope.CorrelationID = &correlationID
	return b
}

// Build construye y valida el envelope
func (b *EnvelopeBuilder) Build() (*Envelope, error) {
	if err := b.envelope.Validate(); err != nil {
		return nil, err
	}
	return b.envelope, nil
}

// Helpers para casos comunes

// RegisterRequest construye un REGISTER
func RegisterRequest(sessionToken, componentName, version string, caps ComponentCapabilities) (*Envelope, error) {
	payload := map[string]interface{}{
		"session_token":  sessionToken,
		"component_name": componentName,
		"version":        version,
		"capabilities":   caps,
	}

	return NewBuilder(MessageTypeLifecycle, CommandRegister, "component", "aegis").
		WithPayload(payload).
		Build()
}

// RegisteredResponse constructs a REGISTERED
func RegisteredResponse(correlationID, componentID, sessionID string) (*Envelope, error) {
	payload := map[string]any{
		"component_id": componentID,
		"session_id":   sessionID,
		"state":        ComponentStateRegistered,
	}

	return NewBuilder(MessageTypeLifecycle, CommandRegistered, "aegis", "component").
		WithCorrelation(correlationID).
		WithPayload(payload).
		Build()
}

// StateUpdateRequest construye un STATE_UPDATE
func StateUpdateRequest(state ComponentState) (*Envelope, error) {
	payload := map[string]any{
		"state": state,
	}

	return NewBuilder(MessageTypeLifecycle, CommandStateUpdate, "component", "aegis").
		WithPayload(payload).
		Build()
}

// ConfigureRequest construye un CONFIGURE
func ConfigureRequest(dataStreamSocket string, topics []string) (*Envelope, error) {
	payload := map[string]interface{}{
		"data_stream_socket": dataStreamSocket,
		"topics":             topics,
	}

	return NewBuilder(MessageTypeConfig, CommandConfigure, "aegis", "component").
		WithPayload(payload).
		Build()
}

// ACKResponse construye un ACK
func ACKResponse(correlationID string) (*Envelope, error) {
	payload := map[string]interface{}{
		"status": "ok",
	}

	return NewBuilder(MessageTypeControl, CommandACK, "component", "aegis").
		WithCorrelation(correlationID).
		WithPayload(payload).
		Build()
}

// PingRequest construye un PING
func PingRequest() (*Envelope, error) {
	return NewBuilder(MessageTypeHeartbeat, CommandPing, "aegis", "component").
		WithPayload(make(map[string]interface{})).
		Build()
}

// PongResponse construye un PONG
func PongResponse(correlationID string, state ComponentState, uptime int64) (*Envelope, error) {
	payload := map[string]interface{}{
		"state":          state,
		"uptime_seconds": uptime,
	}

	return NewBuilder(MessageTypeHeartbeat, CommandPong, "component", "aegis").
		WithCorrelation(correlationID).
		WithPayload(payload).
		Build()
}

// ErrorResponse construye un ERROR
func ErrorResponse(correlationID, code, message string, recoverable bool) (*Envelope, error) {
	payload := map[string]interface{}{
		"code":        code,
		"message":     message,
		"recoverable": recoverable,
	}

	return NewBuilder(MessageTypeError, CommandRuntimeError, "component", "aegis").
		WithCorrelation(correlationID).
		WithPayload(payload).
		Build()
}
