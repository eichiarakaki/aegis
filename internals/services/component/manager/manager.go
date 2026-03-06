package manager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/eichiarakaki/aegis/internals/services/utils"

	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleComponentConnection manages incoming connections from components.
func HandleComponentConnection(conn net.Conn, sessionStore *core.SessionStore, pool *servicescomponent.ConnectionPool) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Errorf("error closing connection: %v", err)
			return
		}
	}(conn)

	logging := logger.WithComponent("ComponentManager").WithField("remote_addr", conn.RemoteAddr().String())
	logging.Debugf("Component connection established")

	dec := json.NewDecoder(bufio.NewReader(conn))

	// STEP 1: Receive REGISTER message
	var registerEnvelope core.Envelope
	if err := dec.Decode(&registerEnvelope); err != nil {
		logging.Errorf("Failed to decode envelope: %s", err.Error())
		sendErrorResponse(conn, "", core.DECODE_ERROR, "Failed to decode message", false)
		return
	}

	if err := registerEnvelope.Validate(); err != nil {
		logging.Warnf("Invalid envelope: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, core.INVALID_ENVELOPE, err.Error(), false)
		return
	}

	if registerEnvelope.Type != core.MessageTypeLifecycle || registerEnvelope.Command != core.CommandRegister {
		logging.Warnf("Expected REGISTER command, got: %s %s", registerEnvelope.Type, registerEnvelope.Command)
		sendErrorResponse(conn, registerEnvelope.MessageID, core.INVALID_COMMAND, "Expected REGISTER command", false)
		return
	}

	var registerPayload core.RegisterPayload
	payloadJSON, _ := json.Marshal(registerEnvelope.Payload)
	if err := json.Unmarshal(payloadJSON, &registerPayload); err != nil {
		logging.Errorf("Failed to parse register payload: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, core.INVALID_PAYLOAD, "Failed to parse payload", false)
		return
	}

	if registerPayload.SessionToken == "" {
		logging.Warnf("Missing session_token in register payload")
		sendErrorResponse(conn, registerEnvelope.MessageID, core.MISSING_SESSION_TOKEN, "session_token is required", false)
		return
	}

	if registerPayload.ComponentName == "" {
		logging.Warnf("Missing component_name in register payload")
		sendErrorResponse(conn, registerEnvelope.MessageID, core.MISSING_COMPONENT_NAME, "component_name is required", false)
		return
	}

	logging = logging.WithField("component_name", registerPayload.ComponentName)
	logging.Infof("Registering component: %s (version: %s)", registerPayload.ComponentName, registerPayload.Version)

	// STEP 2: Validate session token and resolve session
	session, err := sessions.GetSessionByHint(registerPayload.SessionToken, sessionStore)
	if err != nil {
		logging.Errorf("Session token does not match any active session: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, core.MISSING_SESSION_TOKEN, "The token provided does not match any active session.", false)
		return
	}

	registry := session.Registry
	if registry == nil {
		logging.Errorf("Session %s has no initialized registry", session.ID)
		sendErrorResponse(conn, registerEnvelope.MessageID, core.SESSION_REGISTRY_UNAVAILABLE, "Session registry is not initialized.", false)
		return
	}

	logging = logging.WithField("session_id", session.ID)

	// STEP 3: Resolve component ID.
	// If the component was launched by aegisd (via LaunchComponents), it will
	// have received AEGIS_COMPONENT_ID and sent it back in the REGISTER payload.
	// Honouring that ID keeps the log-pump subject (aegis.logs.<id>) and the
	// registry entry in sync. Fall back to generating a new ID only when the
	// component connected manually without a pre-assigned ID.
	componentID := registerPayload.ComponentID
	if componentID == "" {
		componentID = utils.GenerateComponentID()
		logging.Debugf("No component_id in REGISTER payload — generated: %s", componentID)
	} else {
		logging.Debugf("Using pre-assigned component_id from REGISTER payload: %s", componentID)
	}

	comp := &core.Component{
		ID:            componentID,
		Name:          registerPayload.ComponentName,
		Version:       registerPayload.Version,
		SessionID:     session.ID,
		State:         core.ComponentStateRegistered,
		Capabilities:  registerPayload.Capabilities,
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
	}

	if err := registry.Register(comp); err != nil {
		logging.Errorf("Failed to register component: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, core.REGISTRATION_FAILED, err.Error(), false)
		return
	}

	logging = logging.WithField("component_id", componentID)
	logging.Infof("Component registered successfully into session %s", session.ID)
	logging.Debugf("Supported symbols: %v", registerPayload.Capabilities.SupportedSymbols)
	logging.Debugf("Supported timeframes: %v", registerPayload.Capabilities.SupportedTimeframes)
	logging.Debugf("Requires streams: %v", registerPayload.Capabilities.RequiresStreams)

	// STEP 4: Send REGISTERED response
	registeredEnvelope, err := RegisteredResponse(registerEnvelope.MessageID, componentID, session.ID)
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

	// STEP 5: Wait for STATE_UPDATE(INITIALIZING) then STATE_UPDATE(READY)
	if err := WaitForReady(conn, dec, registry, comp, logging); err != nil {
		logging.Errorf("Component did not become READY: %s", err.Error())
		registry.Unregister(componentID)
		return
	}

	// STEP 6: Build and send CONFIGURE
	streamSocketPath := fmt.Sprintf("/tmp/aegis-data-stream-%s.sock", session.ID)
	newTopics := BuildTopics(registerPayload.Capabilities)

	session.StreamSocket = &streamSocketPath
	session.AddTopics(componentID, newTopics)
	defer session.RemoveComponentTopics(componentID, newTopics)

	configureEnvelope, err := ConfigureResponse(componentID, streamSocketPath, newTopics)
	if err != nil {
		logging.Errorf("Failed to create CONFIGURE envelope: %s", err.Error())
		sendErrorResponse(conn, "", core.INTERNAL_ERROR, "Failed to build configuration", false)
		err := registry.Unregister(componentID)
		if err != nil {
			logger.Errorf("Failed to unregister component: %s", err.Error())
			return
		}
		return
	}
	if err := json.NewEncoder(conn).Encode(configureEnvelope); err != nil {
		logging.Errorf("Failed to send CONFIGURE: %s", err.Error())
		err := registry.Unregister(componentID)
		if err != nil {
			logger.Errorf("Failed to unregister component: %s", err.Error())
			return
		}
		return
	}
	logging.Infof("Sent CONFIGURE — socket=%s topics=%v", streamSocketPath, newTopics)

	// STEP 7: Wait for ACK of CONFIGURE
	if err := WaitForConfigACK(conn, dec, configureEnvelope.MessageID, logging); err != nil {
		logging.Errorf("Component did not ACK configuration: %s", err.Error())
		sendErrorResponse(conn, "", core.CONFIG_ACK_TIMEOUT, "Component did not acknowledge configuration", false)
		err := registry.Unregister(componentID)
		if err != nil {
			logger.Errorf("Failed to unregister component: %s", err.Error())
			return
		}
		return
	}
	logging.Infof("Configuration acknowledged by component — handing off to lifecycle loop")

	// STEP 8: Steady-state lifecycle loop
	handleComponentLifecycle(conn, dec, registry, comp, logging)
}

func handleComponentLifecycle(
	conn net.Conn,
	dec *json.Decoder,
	registry *core.Registry,
	comp *core.Component,
	logger *logger.Logger,
) {
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return
	}

	for {
		var envelope core.Envelope
		if err := dec.Decode(&envelope); err != nil {
			logger.Warnf("Connection closed or error reading message: %s", err.Error())
			_ = registry.Unregister(comp.ID)
			logger.Infof("Component unregistered due to connection loss")
			return
		}

		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return
		}

		if err := envelope.Validate(); err != nil {
			logger.Warnf("Invalid envelope: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, core.INVALID_ENVELOPE, err.Error(), false)
			continue
		}

		logger.Debugf("Received message: type=%s command=%s", envelope.Type, envelope.Command)

		switch envelope.Type {
		case core.MessageTypeLifecycle:
			handleLifecycleMessage(conn, registry, comp, &envelope, logger)
		case core.MessageTypeHeartbeat:
			handleHeartbeatMessage(conn, registry, comp, &envelope, logger)
		case core.MessageTypeConfig:
			handleConfigMessage(conn, registry, comp, &envelope, logger)
		default:
			logger.Warnf("Unknown message type: %s", envelope.Type)
			sendErrorResponse(conn, envelope.MessageID, "UNKNOWN_MESSAGE_TYPE", "Unknown message type", false)
		}
	}
}

func handleLifecycleMessage(
	conn net.Conn,
	registry *core.Registry,
	comp *core.Component,
	envelope *core.Envelope,
	logger *logger.Logger,
) {
	switch envelope.Command {
	case core.CommandStateUpdate:
		var payload core.StateUpdatePayload
		payloadJSON, _ := json.Marshal(envelope.Payload)
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			logger.Errorf("Failed to parse state update payload: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, core.INVALID_PAYLOAD, "Failed to parse payload", false)
			return
		}

		if err := registry.UpdateState(comp.ID, payload.State); err != nil {
			logger.Warnf("Failed to update state: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, core.STATE_TRANSITION_FAILED, err.Error(), false)
			return
		}

		logger.Infof("Component state updated to: %s", payload.State)

		ackEnvelope, _ := ACKResponse(envelope.MessageID)
		if err := json.NewEncoder(conn).Encode(ackEnvelope); err != nil {
			return
		}

	case core.CommandShutdown:
		logger.Infof("Component initiated shutdown")
		ackEnvelope, _ := ACKResponse(envelope.MessageID)
		_ = json.NewEncoder(conn).Encode(ackEnvelope)
		_ = registry.Unregister(comp.ID)

	default:
		logger.Warnf("Unknown lifecycle command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, core.UNKNOWN_COMMAND, "Unknown lifecycle command", false)
	}
}

func handleHeartbeatMessage(
	conn net.Conn,
	registry *core.Registry,
	comp *core.Component,
	envelope *core.Envelope,
	logger *logger.Logger,
) {
	switch envelope.Command {
	case core.CommandPing:
		uptimeSeconds := int64(time.Since(comp.StartedAt).Seconds())
		pongEnvelope, _ := PongResponse(envelope.MessageID, comp.State, uptimeSeconds)
		if err := json.NewEncoder(conn).Encode(pongEnvelope); err != nil {
			return
		}
		logger.Debugf("Sent PONG response")

	case core.CommandPong:
		logger.Debugf("Received PONG from component")
		_ = registry.UpdateHeartbeat(comp.ID)

	default:
		logger.Warnf("Unknown heartbeat command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, core.UNKNOWN_COMMAND, "Unknown heartbeat command", false)
	}
}

func handleConfigMessage(
	conn net.Conn,
	registry *core.Registry,
	comp *core.Component,
	envelope *core.Envelope,
	logger *logger.Logger,
) {
	switch envelope.Command {
	case core.CommandConfigure:
		var payload core.ConfigurePayload
		payloadJSON, _ := json.Marshal(envelope.Payload)
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			logger.Errorf("Failed to parse configure payload: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, core.INVALID_PAYLOAD, "Failed to parse payload", false)
			return
		}
		if err := registry.UpdateState(comp.ID, core.ComponentStateConfigured); err != nil {
			logger.Errorf("Failed to update state to CONFIGURED: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, core.STATE_TRANSITION_FAILED, err.Error(), false)
			return
		}
		ackEnvelope, _ := ACKResponse(envelope.MessageID)
		_ = json.NewEncoder(conn).Encode(ackEnvelope)
		logger.Debugf("Configuration acknowledged")

	default:
		logger.Warnf("Unknown config command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, core.UNKNOWN_COMMAND, "Unknown config command", false)
	}
}

func sendErrorResponse(conn net.Conn, correlationID, code core.ErrorCode, message string, recoverable bool) {
	errorEnvelope, err := ErrorResponse(correlationID, code, message, recoverable)
	if err != nil {
		logger.Errorf("Failed to create error response: %s", err.Error())
		return
	}
	if err := json.NewEncoder(conn).Encode(errorEnvelope); err != nil {
		logger.Errorf("Failed to send error response: %s", err.Error())
	}
}
