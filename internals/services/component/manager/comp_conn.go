package manager

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/utils"
)

func RegisteredResponse(correlationID, componentID, sessionID string) (*core.Envelope, error) {
	if componentID == "" || sessionID == "" {
		return nil, fmt.Errorf("componentID and sessionID are required")
	}
	env := core.NewEnvelope(
		core.MessageTypeLifecycle,
		core.CommandRegistered,
		"aegis",
		"component:"+componentID,
		map[string]any{
			"component_id": componentID,
			"session_id":   sessionID,
			"state":        string(core.ComponentStateRegistered),
		},
	)
	env.WithCorrelation(correlationID)
	return env, nil
}

func ConfigureResponse(componentID, streamSocketPath string, topics []string) (*core.Envelope, error) {
	if streamSocketPath == "" {
		return nil, fmt.Errorf("streamSocketPath is required")
	}
	if topics == nil {
		topics = []string{}
	}
	env := core.NewEnvelope(
		core.MessageTypeConfig,
		core.CommandConfigure,
		"aegis",
		"component:"+componentID,
		map[string]interface{}{
			"data_stream_socket": streamSocketPath,
			"topics":             topics,
		},
	)
	return env, nil
}

func ACKResponse(correlationID string) (*core.Envelope, error) {
	env := core.NewEnvelope(
		core.MessageTypeControl,
		core.CommandACK,
		"aegis",
		"component:unknown",
		map[string]interface{}{"status": "ok"},
	)
	env.WithCorrelation(correlationID)
	return env, nil
}

func PongResponse(correlationID string, state core.ForeignComponentState, uptimeSeconds int64) (*core.Envelope, error) {
	env := core.NewEnvelope(
		core.MessageTypeHeartbeat,
		core.CommandPong,
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

func ErrorResponse(correlationID, code, message string, recoverable bool) (*core.Envelope, error) {
	env := core.NewEnvelope(
		core.MessageTypeError,
		core.CommandRuntimeError,
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

// WaitForReady reads STATE_UPDATE(INITIALIZING) then STATE_UPDATE(READY),
// ACKing each one. Uses the shared decoder to avoid consuming bytes from
// the connection's internal buffer.
func WaitForReady(
	conn net.Conn,
	dec *json.Decoder,
	registry *core.Registry,
	comp *core.Component,
	log *logger.Logger,
) error {
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	defer conn.SetReadDeadline(time.Time{})

	expected := []core.ForeignComponentState{
		core.ComponentStateInitializing,
		core.ComponentStateReady,
	}

	for _, expectedState := range expected {
		log.Debugf("Waiting for STATE_UPDATE(%s)…", expectedState)

		var envelope core.Envelope
		if err := dec.Decode(&envelope); err != nil {
			return fmt.Errorf("failed to read STATE_UPDATE(%s): %w", expectedState, err)
		}
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}

		if err := envelope.Validate(); err != nil {
			sendErrorResponse(conn, envelope.MessageID, core.INVALID_ENVELOPE, err.Error(), false)
			return fmt.Errorf("invalid envelope while waiting for %s: %w", expectedState, err)
		}

		if envelope.Type != core.MessageTypeLifecycle || envelope.Command != core.CommandStateUpdate {
			sendErrorResponse(conn, envelope.MessageID, core.UNEXPECTED_MESSAGE,
				fmt.Sprintf("Expected STATE_UPDATE(%s)", expectedState), false)
			return fmt.Errorf("unexpected message while waiting for %s: type=%s command=%s",
				expectedState, envelope.Type, envelope.Command)
		}

		var payload core.StateUpdatePayload
		payloadJSON, _ := json.Marshal(envelope.Payload)
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			sendErrorResponse(conn, envelope.MessageID, core.INVALID_PAYLOAD, "Failed to parse state update payload", false)
			return fmt.Errorf("failed to parse state update payload: %w", err)
		}

		if payload.State != expectedState {
			sendErrorResponse(conn, envelope.MessageID, core.UNEXPECTED_STATE,
				fmt.Sprintf("Expected %s, got %s", expectedState, payload.State), false)
			return fmt.Errorf("expected %s state, got %s", expectedState, payload.State)
		}

		if err := registry.UpdateState(comp.ID, expectedState); err != nil {
			sendErrorResponse(conn, envelope.MessageID, core.STATE_TRANSITION_FAILED, err.Error(), false)
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

// WaitForConfigACK reads the ACK for the CONFIGURE message using the shared decoder.
func WaitForConfigACK(
	conn net.Conn,
	dec *json.Decoder,
	configureMessageID string,
	logger *logger.Logger,
) error {
	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	defer func(conn net.Conn, t time.Time) {
		err := conn.SetReadDeadline(t)
		if err != nil {
			logger.Errorf("error closing connection: %v", err)
			return
		}
	}(conn, time.Time{})

	logger.Debugf("Waiting for config ACK (correlating to message_id=%s)…", configureMessageID)

	var envelope core.Envelope
	if err := dec.Decode(&envelope); err != nil {
		return fmt.Errorf("failed to read ACK: %w", err)
	}

	if err := envelope.Validate(); err != nil {
		return fmt.Errorf("invalid ACK envelope: %w", err)
	}

	if envelope.Command != core.CommandACK {
		return fmt.Errorf("expected ACK for CONFIGURE, got type=%s command=%s",
			envelope.Type, envelope.Command)
	}

	if envelope.CorrelationID == nil || *envelope.CorrelationID != configureMessageID {
		logger.Warnf("ACK correlation mismatch: expected=%s got=%v",
			configureMessageID, envelope.CorrelationID)
	}

	logger.Debugf("Config ACK received")
	return nil
}

func NewMessageID() string {
	return utils.GenerateSecureToken()
}

func Rfc3339Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func BuildTopics(caps core.ComponentCapabilities) []string {
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
