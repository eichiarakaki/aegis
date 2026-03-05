package manager

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

// RegisteredResponse builds the LIFECYCLE/REGISTERED response sent after a
// successful component registration.
func RegisteredResponse(correlationID, componentID, sessionID string) (*component.Envelope, error) {
	if componentID == "" || sessionID == "" {
		return nil, fmt.Errorf("componentID and sessionID are required")
	}

	env := component.NewEnvelope(
		component.MessageTypeLifecycle,
		component.CommandRegistered,
		"aegis",
		"component:"+componentID,
		map[string]any{
			"component_id": componentID,
			"session_id":   sessionID,
			"state":        string(component.ComponentStateRegistered),
		},
	)
	env.WithCorrelation(correlationID)
	return env, nil
}

// ConfigureResponse builds a CONFIG/CONFIGURE envelope addressed to a specific component.
func ConfigureResponse(componentID, streamSocketPath string, topics []string) (*component.Envelope, error) {
	if streamSocketPath == "" {
		return nil, fmt.Errorf("streamSocketPath is required")
	}
	if topics == nil {
		topics = []string{}
	}

	env := component.NewEnvelope(
		component.MessageTypeConfig,
		component.CommandConfigure,
		"aegis",
		"component:"+componentID,
		map[string]interface{}{
			"data_stream_socket": streamSocketPath,
			"topics":             topics,
		},
	)
	return env, nil
}

// ACKResponse builds a CONTROL/ACK envelope correlated to the given message.
func ACKResponse(correlationID string) (*component.Envelope, error) {
	env := component.NewEnvelope(
		component.MessageTypeControl,
		component.CommandACK,
		"aegis",
		"component:unknown", // target is overridden by the caller's conn
		map[string]interface{}{
			"status": "ok",
		},
	)
	env.WithCorrelation(correlationID)
	return env, nil
}

// PongResponse builds a HEARTBEAT/PONG envelope correlated to a PING.
func PongResponse(correlationID string, state component.ComponentState, uptimeSeconds int64) (*component.Envelope, error) {
	env := component.NewEnvelope(
		component.MessageTypeHeartbeat,
		component.CommandPong,
		"aegis",
		"component:unknown",
		map[string]interface{}{
			"state":          string(state),
			"uptime_seconds": uptimeSeconds,
		},
	)
	env.WithCorrelation(correlationID)
	return env, nil
}

func WaitForConfigACK(
	conn net.Conn,
	configureMessageID string,
	logger *logger.Logger,
) error {
	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	defer conn.SetReadDeadline(time.Time{})

	logger.Debugf("Waiting for config ACK (correlating to message_id=%s)…", configureMessageID)

	var envelope component.Envelope
	if err := json.NewDecoder(conn).Decode(&envelope); err != nil {
		return fmt.Errorf("failed to read ACK: %w", err)
	}

	if err := envelope.Validate(); err != nil {
		return fmt.Errorf("invalid ACK envelope: %w", err)
	}

	if envelope.Command != component.CommandACK {
		return fmt.Errorf(
			"expected ACK for CONFIGURE, got type=%s command=%s",
			envelope.Type, envelope.Command,
		)
	}

	// CorrelationID is *string — dereference safely
	if envelope.CorrelationID == nil || *envelope.CorrelationID != configureMessageID {
		logger.Warnf(
			"ACK correlation mismatch: expected=%s got=%v",
			configureMessageID, envelope.CorrelationID,
		)
	}

	logger.Debugf("Config ACK received")
	return nil
}

// newMessageID is a convenience wrapper so response builders don't need to
// import utils directly.
func NewMessageID() string {
	return utils.GenerateSecureToken()
}

// rfc3339Now returns the current UTC time formatted as RFC3339.
func Rfc3339Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// waitForReady blocks until the component sends STATE_UPDATE(READY), ACKs it,
// and updates the registry. Any other message type or state is rejected.
func WaitForReady(
	conn net.Conn,
	registry *component.ComponentRegistry,
	comp *component.Component,
	log *logger.Logger,
) error {
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	defer conn.SetReadDeadline(time.Time{})

	// The client walks REGISTERED → INITIALIZING → READY.
	// We accept both state updates here, ACKing each one, before proceeding.
	expected := []component.ComponentState{
		component.ComponentStateInitializing,
		component.ComponentStateReady,
	}

	for _, expectedState := range expected {
		log.Debugf("Waiting for STATE_UPDATE(%s)…", expectedState)

		var envelope component.Envelope
		if err := json.NewDecoder(conn).Decode(&envelope); err != nil {
			return fmt.Errorf("failed to read STATE_UPDATE(%s): %w", expectedState, err)
		}

		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}

		if err := envelope.Validate(); err != nil {
			sendErrorResponse(conn, envelope.MessageID, "INVALID_ENVELOPE", err.Error(), false)
			return fmt.Errorf("invalid envelope while waiting for %s: %w", expectedState, err)
		}

		if envelope.Type != component.MessageTypeLifecycle || envelope.Command != component.CommandStateUpdate {
			sendErrorResponse(conn, envelope.MessageID, "UNEXPECTED_MESSAGE",
				fmt.Sprintf("Expected STATE_UPDATE(%s)", expectedState), false)
			return fmt.Errorf("unexpected message while waiting for %s: type=%s command=%s",
				expectedState, envelope.Type, envelope.Command)
		}

		var payload component.StateUpdatePayload
		payloadJSON, _ := json.Marshal(envelope.Payload)
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			sendErrorResponse(conn, envelope.MessageID, "INVALID_PAYLOAD", "Failed to parse state update payload", false)
			return fmt.Errorf("failed to parse state update payload: %w", err)
		}

		if payload.State != expectedState {
			sendErrorResponse(conn, envelope.MessageID, "UNEXPECTED_STATE",
				fmt.Sprintf("Expected %s, got %s", expectedState, payload.State), false)
			return fmt.Errorf("expected %s state, got %s", expectedState, payload.State)
		}

		if err := registry.UpdateState(comp.ID, expectedState); err != nil {
			sendErrorResponse(conn, envelope.MessageID, "STATE_TRANSITION_FAILED", err.Error(), false)
			return fmt.Errorf("failed to update state to %s: %w", expectedState, err)
		}

		ackEnvelope, err := ACKResponse(envelope.MessageID)
		if err != nil {
			return fmt.Errorf("failed to create ACK: %w", err)
		}
		if err := json.NewEncoder(conn).Encode(ackEnvelope); err != nil {
			return fmt.Errorf("failed to send ACK for %s: %w", expectedState, err)
		}

		log.Infof("Component transitioned to %s", expectedState)
	}

	return nil
}

// ErrorResponse builds an ERROR envelope with the given code and message.
func ErrorResponse(correlationID, code, message string, recoverable bool) (*component.Envelope, error) {
	env := component.NewEnvelope(
		component.MessageTypeError,
		component.CommandRuntimeError,
		"aegis",
		"component:unknown",
		map[string]any{
			"code":        code,
			"message":     message,
			"recoverable": recoverable,
		},
	)
	if correlationID != "" {
		env.WithCorrelation(correlationID)
	}
	return env, nil
}

// buildTopics derives the list of data-stream topic strings from the
// component's declared capabilities.
func BuildTopics(caps component.ComponentCapabilities) []string {
	timeframedStreams := map[string]bool{
		"klines": true,
	}

	var topics []string
	for _, stream := range caps.RequiresStreams {
		if timeframedStreams[stream] {
			for _, symbol := range caps.SupportedSymbols {
				for _, tf := range caps.SupportedTimeframes {
					topics = append(topics, stream+"."+symbol+"."+tf)
				}
			}
		} else {
			for _, symbol := range caps.SupportedSymbols {
				topics = append(topics, stream+"."+symbol)
			}
		}
	}
	return topics
}
