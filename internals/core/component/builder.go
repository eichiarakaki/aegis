package component

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/logger"
)

// RegisteredResponse builds the LIFECYCLE/REGISTERED response sent after a
// successful component registration.
func RegisteredResponse(correlationID, componentID, sessionID string) (*Envelope, error) {
	if componentID == "" || sessionID == "" {
		return nil, fmt.Errorf("componentID and sessionID are required")
	}

	env := NewEnvelope(
		MessageTypeLifecycle,
		CommandRegistered,
		"aegis",
		"component:"+componentID,
		map[string]interface{}{
			"component_id": componentID,
			"session_id":   sessionID,
			"state":        string(ComponentStateRegistered),
		},
	)
	env.WithCorrelation(correlationID)
	return env, nil
}

// ConfigureResponse builds a CONFIG/CONFIGURE envelope addressed to a specific component.
func ConfigureResponse(componentID, streamSocketPath string, topics []string) (*Envelope, error) {
	if streamSocketPath == "" {
		return nil, fmt.Errorf("streamSocketPath is required")
	}
	if topics == nil {
		topics = []string{}
	}

	env := NewEnvelope(
		MessageTypeConfig,
		CommandConfigure,
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
func ACKResponse(correlationID string) (*Envelope, error) {
	env := NewEnvelope(
		MessageTypeControl,
		CommandACK,
		"aegis",
		"component:unknown",
		map[string]interface{}{
			"status": "ok",
		},
	)
	env.WithCorrelation(correlationID)
	return env, nil
}

// PongResponse builds a HEARTBEAT/PONG envelope correlated to a PING.
func PongResponse(correlationID string, state ComponentState, uptimeSeconds int64) (*Envelope, error) {
	env := NewEnvelope(
		MessageTypeHeartbeat,
		CommandPong,
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

// ErrorResponse builds an ERROR envelope with the given code and message.
func ErrorResponse(correlationID, code, message string, recoverable bool) (*Envelope, error) {
	env := NewEnvelope(
		MessageTypeError,
		CommandRuntimeError,
		"aegis",
		"component:unknown",
		map[string]interface{}{
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

// waitForConfigACK blocks until the component sends a CONTROL/ACK correlated
// to the CONFIGURE message that was just sent.
func WaitForConfigACK(
	conn net.Conn,
	configureMessageID string,
	log *logger.Logger,
) error {
	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	defer conn.SetReadDeadline(time.Time{})

	log.Debugf("Waiting for config ACK (correlating to message_id=%s)…", configureMessageID)

	var envelope Envelope
	if err := json.NewDecoder(conn).Decode(&envelope); err != nil {
		return fmt.Errorf("failed to read ACK: %w", err)
	}

	if err := envelope.Validate(); err != nil {
		return fmt.Errorf("invalid ACK envelope: %w", err)
	}

	if envelope.Command != CommandACK {
		return fmt.Errorf(
			"expected ACK for CONFIGURE, got type=%s command=%s",
			envelope.Type, envelope.Command,
		)
	}

	if envelope.CorrelationID == nil || *envelope.CorrelationID != configureMessageID {
		log.Warnf(
			"ACK correlation mismatch: expected=%s got=%v",
			configureMessageID, envelope.CorrelationID,
		)
	}

	log.Debugf("Config ACK received")
	return nil
}

// buildTopics derives the list of data-stream topic strings from the
// component's declared capabilities.
func BuildTopics(caps ComponentCapabilities) []string {
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
