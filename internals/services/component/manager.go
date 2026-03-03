package component

import (
	"encoding/json"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/google/uuid"
)

// HandleComponentConnection manages incoming connections from components.
// It handles the registration handshake, associates the component with the
// correct session registry, and manages the component lifecycle.
func HandleComponentConnection(conn net.Conn, sessionStore *core.SessionStore, pool *ConnectionPool) {
	defer func(conn net.Conn) {
		if err := conn.Close(); err != nil {
			return
		}
	}(conn)

	logging := logger.WithComponent("ComponentManager").WithField("remote_addr", conn.RemoteAddr().String())
	logging.Debugf("Component connection established")

	// STEP 1: Receive REGISTER message
	var registerEnvelope component.Envelope
	if err := json.NewDecoder(conn).Decode(&registerEnvelope); err != nil {
		logging.Errorf("Failed to decode envelope: %s", err.Error())
		sendErrorResponse(conn, "", "DECODE_ERROR", "Failed to decode message", false)
		return
	}

	// Validate envelope structure
	if err := registerEnvelope.Validate(); err != nil {
		logging.Warnf("Invalid envelope: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, "INVALID_ENVELOPE", err.Error(), false)
		return
	}

	// Validate it's a REGISTER command
	if registerEnvelope.Type != component.MessageTypeLifecycle || registerEnvelope.Command != component.CommandRegister {
		logging.Warnf("Expected REGISTER command, got: %s %s", registerEnvelope.Type, registerEnvelope.Command)
		sendErrorResponse(conn, registerEnvelope.MessageID, "INVALID_COMMAND", "Expected REGISTER command", false)
		return
	}

	// Parse register payload
	var registerPayload component.RegisterPayload
	payloadJSON, _ := json.Marshal(registerEnvelope.Payload)
	if err := json.Unmarshal(payloadJSON, &registerPayload); err != nil {
		logging.Errorf("Failed to parse register payload: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, "INVALID_PAYLOAD", "Failed to parse payload", false)
		return
	}

	// Validate required fields
	if registerPayload.SessionToken == "" {
		logging.Warnf("Missing session_token in register payload")
		sendErrorResponse(conn, registerEnvelope.MessageID, "MISSING_SESSION_TOKEN", "session_token is required", false)
		return
	}

	if registerPayload.ComponentName == "" {
		logging.Warnf("Missing component_name in register payload")
		sendErrorResponse(conn, registerEnvelope.MessageID, "MISSING_COMPONENT_NAME", "component_name is required", false)
		return
	}

	logging = logging.WithField("component_name", registerPayload.ComponentName)
	logging.Infof("Registering component: %s (version: %s)", registerPayload.ComponentName, registerPayload.Version)

	// STEP 2: Validate session token and resolve session
	session, err := sessions.GetSessionByHint(registerPayload.SessionToken, sessionStore)
	if err != nil {
		logging.Errorf("Session token does not match any active session: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, "WRONG_SESSION_TOKEN", "The token provided does not match any active session.", false)
		return
	}

	// Use the registry belonging to the resolved session
	registry := session.Registry
	if registry == nil {
		logging.Errorf("Session %s has no initialized registry", session.ID)
		sendErrorResponse(conn, registerEnvelope.MessageID, "SESSION_REGISTRY_UNAVAILABLE", "Session registry is not initialized.", false)
		return
	}

	logging = logging.WithField("session_id", session.ID)

	// STEP 3: Create component record
	componentID := "cmp-" + uuid.New().String()[:8]
	comp := &component.Component{
		ID:            componentID,
		Name:          registerPayload.ComponentName,
		Version:       registerPayload.Version,
		SessionID:     session.ID,
		State:         component.ComponentStateRegistered,
		Capabilities:  registerPayload.Capabilities,
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
	}

	// Register the component into the session's own registry
	if err := registry.Register(comp); err != nil {
		logging.Errorf("Failed to register component: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, "REGISTRATION_FAILED", err.Error(), false)
		return
	}

	logging = logging.WithField("component_id", componentID)
	logging.Infof("Component registered successfully into session %s", session.ID)

	// Log capabilities
	logging.Debugf("Supported symbols: %v", registerPayload.Capabilities.SupportedSymbols)
	logging.Debugf("Supported timeframes: %v", registerPayload.Capabilities.SupportedTimeframes)
	logging.Debugf("Requires streams: %v", registerPayload.Capabilities.RequiresStreams)

	// STEP 4: Send REGISTERED response
	registeredEnvelope, err := component.RegisteredResponse(
		registerEnvelope.MessageID,
		componentID,
		session.ID,
	)
	if err != nil {
		logging.Errorf("Failed to create registered response: %s", err.Error())
		return
	}

	if err := json.NewEncoder(conn).Encode(registeredEnvelope); err != nil {
		logging.Errorf("Failed to send registered response: %s", err.Error())
		return
	}

	logging.Debugf("Sent REGISTERED response")

	pool.Add(componentID, conn)
	defer pool.Remove(componentID)

	// STEP 5: Handle component lifecycle messages
	handleComponentLifecycle(conn, registry, comp, logging)
}

// handleComponentLifecycle manages the component's lifecycle messages and heartbeats.
func handleComponentLifecycle(
	conn net.Conn,
	registry *component.ComponentRegistry,
	comp *component.Component,
	logger *logger.Logger,
) {
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return
	}

	for {
		var envelope component.Envelope
		if err := json.NewDecoder(conn).Decode(&envelope); err != nil {
			logger.Warnf("Connection closed or error reading message: %s", err.Error())
			_ = registry.Unregister(comp.ID)
			logger.Infof("Component unregistered due to connection loss")
			return
		}

		// Reset read deadline after each successful read
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return
		}

		if err := envelope.Validate(); err != nil {
			logger.Warnf("Invalid envelope: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, "INVALID_ENVELOPE", err.Error(), false)
			continue
		}

		logger.Debugf("Received message: type=%s command=%s", envelope.Type, envelope.Command)

		switch envelope.Type {
		case component.MessageTypeLifecycle:
			handleLifecycleMessage(conn, registry, comp, &envelope, logger)

		case component.MessageTypeHeartbeat:
			handleHeartbeatMessage(conn, registry, comp, &envelope, logger)

		case component.MessageTypeConfig:
			handleConfigMessage(conn, registry, comp, &envelope, logger)

		default:
			logger.Warnf("Unknown message type: %s", envelope.Type)
			sendErrorResponse(conn, envelope.MessageID, "UNKNOWN_MESSAGE_TYPE", "Unknown message type", false)
		}
	}
}

// handleLifecycleMessage processes lifecycle state update messages.
func handleLifecycleMessage(
	conn net.Conn,
	registry *component.ComponentRegistry,
	comp *component.Component,
	envelope *component.Envelope,
	logger *logger.Logger,
) {
	switch envelope.Command {
	case component.CommandStateUpdate:
		var payload component.StateUpdatePayload
		payloadJSON, _ := json.Marshal(envelope.Payload)
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			logger.Errorf("Failed to parse state update payload: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, "INVALID_PAYLOAD", "Failed to parse payload", false)
			return
		}

		if err := registry.UpdateState(comp.ID, payload.State); err != nil {
			logger.Warnf("Failed to update state: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, "STATE_TRANSITION_FAILED", err.Error(), false)
			return
		}

		logger.Infof("Component state updated to: %s", payload.State)

		ackEnvelope, _ := component.ACKResponse(envelope.MessageID)
		if err := json.NewEncoder(conn).Encode(ackEnvelope); err != nil {
			return
		}

	case component.CommandShutdown:
		logger.Infof("Component initiated shutdown")

		ackEnvelope, _ := component.ACKResponse(envelope.MessageID)
		if err := json.NewEncoder(conn).Encode(ackEnvelope); err != nil {
			return
		}

		if err := registry.Unregister(comp.ID); err != nil {
			return
		}

	default:
		logger.Warnf("Unknown lifecycle command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, "UNKNOWN_COMMAND", "Unknown lifecycle command", false)
	}
}

// handleHeartbeatMessage processes heartbeat messages (PING/PONG).
func handleHeartbeatMessage(
	conn net.Conn,
	registry *component.ComponentRegistry,
	comp *component.Component,
	envelope *component.Envelope,
	logger *logger.Logger,
) {
	switch envelope.Command {
	case component.CommandPing:
		uptimeSeconds := int64(time.Since(comp.StartedAt).Seconds())
		pongEnvelope, _ := component.PongResponse(envelope.MessageID, comp.State, uptimeSeconds)
		if err := json.NewEncoder(conn).Encode(pongEnvelope); err != nil {
			return
		}
		logger.Debugf("Sent PONG response")

	case component.CommandPong:
		logger.Debugf("Received PONG from component")
		if err := registry.UpdateHeartbeat(comp.ID); err != nil {
			return
		}

	default:
		logger.Warnf("Unknown heartbeat command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, "UNKNOWN_COMMAND", "Unknown heartbeat command", false)
	}
}

// handleConfigMessage processes configuration messages.
func handleConfigMessage(
	conn net.Conn,
	registry *component.ComponentRegistry,
	comp *component.Component,
	envelope *component.Envelope,
	logger *logger.Logger,
) {
	switch envelope.Command {
	case component.CommandConfigure:
		var payload component.ConfigurePayload
		payloadJSON, _ := json.Marshal(envelope.Payload)
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			logger.Errorf("Failed to parse configure payload: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, "INVALID_PAYLOAD", "Failed to parse payload", false)
			return
		}

		logger.Infof("Received configuration: data_stream_socket=%s topics=%v", payload.DataStreamSocket, payload.Topics)

		// TODO: Apply configuration to component

		if err := registry.UpdateState(comp.ID, component.ComponentStateConfigured); err != nil {
			logger.Errorf("Failed to update state to CONFIGURED: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, "STATE_TRANSITION_FAILED", err.Error(), false)
			return
		}

		ackEnvelope, _ := component.ACKResponse(envelope.MessageID)
		if err := json.NewEncoder(conn).Encode(ackEnvelope); err != nil {
			return
		}
		logger.Debugf("Configuration acknowledged")

	default:
		logger.Warnf("Unknown config command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, "UNKNOWN_COMMAND", "Unknown config command", false)
	}
}

// sendErrorResponse sends an error response envelope over the connection.
func sendErrorResponse(
	conn net.Conn,
	correlationID string,
	code string,
	message string,
	recoverable bool,
) {
	errorEnvelope, err := component.ErrorResponse(correlationID, code, message, recoverable)
	if err != nil {
		logger.Errorf("Failed to create error response: %s", err.Error())
		return
	}

	if err := json.NewEncoder(conn).Encode(errorEnvelope); err != nil {
		logger.Errorf("Failed to send error response: %s", err.Error())
	}
}
