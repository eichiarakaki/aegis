package manager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/utils"
)

// HandleComponentConnection manages incoming connections from components.
func HandleComponentConnection(conn net.Conn, sessionStore *core.SessionStore, pool *servicescomponent.ConnectionPool) {
	defer conn.Close()

	logging := logger.WithComponent("ComponentManager").WithField("remote_addr", conn.RemoteAddr().String())
	logging.Debugf("Component connection established")

	dec := json.NewDecoder(bufio.NewReader(conn))

	// STEP 1: Receive REGISTER
	var registerEnvelope core.Envelope
	if err := dec.Decode(&registerEnvelope); err != nil {
		logging.Errorf("Failed to decode envelope: %s", err)
		sendErrorResponse(conn, "", core.DECODE_ERROR, "Failed to decode message", false)
		return
	}
	if err := registerEnvelope.Validate(); err != nil {
		logging.Warnf("Invalid envelope: %s", err)
		sendErrorResponse(conn, registerEnvelope.MessageID, core.INVALID_ENVELOPE, err.Error(), false)
		return
	}
	if registerEnvelope.Type != core.MessageTypeLifecycle || registerEnvelope.Command != core.CommandRegister {
		logging.Warnf("Expected REGISTER, got: %s %s", registerEnvelope.Type, registerEnvelope.Command)
		sendErrorResponse(conn, registerEnvelope.MessageID, core.INVALID_COMMAND, "Expected REGISTER command", false)
		return
	}

	var registerPayload core.RegisterPayload
	payloadJSON, _ := json.Marshal(registerEnvelope.Payload)
	if err := json.Unmarshal(payloadJSON, &registerPayload); err != nil {
		logging.Errorf("Failed to parse register payload: %s", err)
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

	// STEP 2: Resolve session
	session, err := sessions.GetSessionByHint(registerPayload.SessionToken, sessionStore)
	if err != nil {
		logging.Errorf("Session not found: %s", err)
		sendErrorResponse(conn, registerEnvelope.MessageID, core.MISSING_SESSION_TOKEN, "The token provided does not match any active session.", false)
		return
	}
	registry := session.Registry
	if registry == nil {
		logging.Errorf("Session %s has no registry", session.ID)
		sendErrorResponse(conn, registerEnvelope.MessageID, core.SESSION_REGISTRY_UNAVAILABLE, "Session registry is not initialized.", false)
		return
	}
	logging = logging.WithField("session_id", session.ID)

	// STEP 3: Resolve or create component entry
	//
	// Four cases:
	//   A. Pre-assigned ID + placeholder in INIT   → hydrate placeholder
	//   B. Pre-assigned ID + existing entry not INIT (reconnect after crash) → reset and reuse
	//   C. Pre-assigned ID + no entry at all       → register fresh
	//   D. No pre-assigned ID (manual connect)     → generate ID and register fresh
	componentID := registerPayload.ComponentID
	var comp *core.Component

	if componentID != "" {
		existing, exists := registry.Get(componentID)
		switch {
		case exists && existing.State == core.ComponentStateInit:
			// Case A: hydrate placeholder created during attach
			if err := registry.UpdateFromRegister(componentID, registerPayload.ComponentName, registerPayload.Version, registerPayload.Capabilities); err != nil {
				logging.Errorf("Failed to hydrate placeholder: %s", err)
				sendErrorResponse(conn, registerEnvelope.MessageID, core.REGISTRATION_FAILED, err.Error(), false)
				return
			}
			if err := registry.UpdateState(componentID, core.ComponentStateRegistered); err != nil {
				logging.Errorf("Failed to transition INIT→REGISTERED: %s", err)
				sendErrorResponse(conn, registerEnvelope.MessageID, core.REGISTRATION_FAILED, err.Error(), false)
				return
			}
			comp, _ = registry.Get(componentID)
			logging.Debugf("Hydrated placeholder: %s", componentID)

		case exists:
			// Case B: component reconnecting after a crash/disconnect — reset state
			logging.Infof("Component reconnecting (was %s) — resetting to REGISTERED", existing.State)
			if err := registry.UpdateFromRegister(componentID, registerPayload.ComponentName, registerPayload.Version, registerPayload.Capabilities); err != nil {
				logging.Errorf("Failed to update reconnecting component: %s", err)
				sendErrorResponse(conn, registerEnvelope.MessageID, core.REGISTRATION_FAILED, err.Error(), false)
				return
			}
			if err := registry.ResetToRegistered(componentID); err != nil {
				logging.Errorf("Failed to reset component state: %s", err)
				sendErrorResponse(conn, registerEnvelope.MessageID, core.REGISTRATION_FAILED, err.Error(), false)
				return
			}
			comp, _ = registry.Get(componentID)

		default:
			// Case C: pre-assigned ID but no entry — register fresh
			comp = &core.Component{
				ID:           componentID,
				Name:         registerPayload.ComponentName,
				Version:      registerPayload.Version,
				SessionID:    session.ID,
				State:        core.ComponentStateRegistered,
				Capabilities: registerPayload.Capabilities,
			}
			if err := registry.Register(comp); err != nil {
				logging.Errorf("Failed to register component: %s", err)
				sendErrorResponse(conn, registerEnvelope.MessageID, core.REGISTRATION_FAILED, err.Error(), false)
				return
			}
			logging.Debugf("Registered fresh (no placeholder found): %s", componentID)
		}
	} else {
		// Case D: manual connect — generate ID
		componentID = utils.GenerateComponentID()
		logging.Debugf("No component_id in REGISTER — generated: %s", componentID)
		comp = &core.Component{
			ID:           componentID,
			Name:         registerPayload.ComponentName,
			Version:      registerPayload.Version,
			SessionID:    session.ID,
			State:        core.ComponentStateRegistered,
			Capabilities: registerPayload.Capabilities,
		}
		if err := registry.Register(comp); err != nil {
			logging.Errorf("Failed to register component: %s", err)
			sendErrorResponse(conn, registerEnvelope.MessageID, core.REGISTRATION_FAILED, err.Error(), false)
			return
		}
	}

	logging = logging.WithField("component_id", componentID)
	logging.Infof("Registering component: %s (version: %s)", registerPayload.ComponentName, registerPayload.Version)
	logging.Debugf("Supported symbols: %v", registerPayload.Capabilities.SupportedSymbols)
	logging.Debugf("Supported timeframes: %v", registerPayload.Capabilities.SupportedTimeframes)
	logging.Debugf("Requires streams: %v", registerPayload.Capabilities.RequiresStreams)

	// STEP 4: Send REGISTERED
	registeredEnvelope, err := RegisteredResponse(registerEnvelope.MessageID, componentID, session.ID)
	if err != nil {
		logging.Errorf("Failed to create REGISTERED response: %s", err)
		return
	}
	if err := json.NewEncoder(conn).Encode(registeredEnvelope); err != nil {
		logging.Errorf("Failed to send REGISTERED: %s", err)
		return
	}
	logging.Debugf("Sent REGISTERED response")

	pool.Add(componentID, conn)
	defer pool.Remove(componentID)

	// STEP 5: Wait for STATE_UPDATE(INITIALIZING) then STATE_UPDATE(READY)
	if err := WaitForReady(conn, dec, registry, comp, logging); err != nil {
		logging.Errorf("Component did not become READY: %s", err)
		_ = registry.Unregister(componentID)
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
		logging.Errorf("Failed to create CONFIGURE: %s", err)
		sendErrorResponse(conn, "", core.INTERNAL_ERROR, "Failed to build configuration", false)
		_ = registry.Unregister(componentID)
		return
	}
	if err := json.NewEncoder(conn).Encode(configureEnvelope); err != nil {
		logging.Errorf("Failed to send CONFIGURE: %s", err)
		_ = registry.Unregister(componentID)
		return
	}
	logging.Infof("Sent CONFIGURE — socket=%s topics=%v", streamSocketPath, newTopics)

	// STEP 7: Wait for ACK of CONFIGURE
	if err := WaitForConfigACK(conn, dec, configureEnvelope.MessageID, logging); err != nil {
		logging.Errorf("Component did not ACK configuration: %s", err)
		sendErrorResponse(conn, "", core.CONFIG_ACK_TIMEOUT, "Component did not acknowledge configuration", false)
		_ = registry.Unregister(componentID)
		return
	}
	logging.Infof("Configuration acknowledged — handing off to lifecycle loop")

	// STEP 8: Steady-state lifecycle loop
	handleComponentLifecycle(conn, dec, registry, comp, logging)
}

func handleComponentLifecycle(
	conn net.Conn,
	dec *json.Decoder,
	registry *core.Registry,
	comp *core.Component,
	log *logger.Logger,
) {
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return
	}

	for {
		var envelope core.Envelope
		if err := dec.Decode(&envelope); err != nil {
			log.Warnf("Connection closed or error: %s", err)
			_ = registry.Unregister(comp.ID)
			log.Infof("Component unregistered due to connection loss")
			return
		}
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return
		}
		if err := envelope.Validate(); err != nil {
			log.Warnf("Invalid envelope: %s", err)
			sendErrorResponse(conn, envelope.MessageID, core.INVALID_ENVELOPE, err.Error(), false)
			continue
		}

		log.Debugf("Received message: type=%s command=%s", envelope.Type, envelope.Command)

		switch envelope.Type {
		case core.MessageTypeLifecycle:
			handleLifecycleMessage(conn, registry, comp, &envelope, log)
		case core.MessageTypeHeartbeat:
			handleHeartbeatMessage(conn, registry, comp, &envelope, log)
		case core.MessageTypeConfig:
			handleConfigMessage(conn, registry, comp, &envelope, log)
		default:
			log.Warnf("Unknown message type: %s", envelope.Type)
			sendErrorResponse(conn, envelope.MessageID, "UNKNOWN_MESSAGE_TYPE", "Unknown message type", false)
		}
	}
}

func handleLifecycleMessage(
	conn net.Conn,
	registry *core.Registry,
	comp *core.Component,
	envelope *core.Envelope,
	log *logger.Logger,
) {
	switch envelope.Command {
	case core.CommandStateUpdate:
		var payload core.StateUpdatePayload
		payloadJSON, _ := json.Marshal(envelope.Payload)
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			log.Errorf("Failed to parse state update payload: %s", err)
			sendErrorResponse(conn, envelope.MessageID, core.INVALID_PAYLOAD, "Failed to parse payload", false)
			return
		}
		if err := registry.UpdateState(comp.ID, payload.State); err != nil {
			log.Warnf("Failed to update state: %s", err)
			sendErrorResponse(conn, envelope.MessageID, core.STATE_TRANSITION_FAILED, err.Error(), false)
			return
		}
		log.Infof("Component state updated to: %s", payload.State)

		// Reset the heartbeat clock when the component becomes RUNNING so the
		// monitor doesn't time it out based on when the placeholder was created.
		if payload.State == core.ComponentStateRunning {
			_ = registry.RefreshHeartbeat(comp.ID)
		}

		ackEnvelope, _ := ACKResponse(envelope.MessageID)
		_ = json.NewEncoder(conn).Encode(ackEnvelope)

	case core.CommandShutdown:
		log.Infof("Component initiated shutdown")
		ackEnvelope, _ := ACKResponse(envelope.MessageID)
		_ = json.NewEncoder(conn).Encode(ackEnvelope)
		_ = registry.Unregister(comp.ID)

	default:
		log.Warnf("Unknown lifecycle command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, core.UNKNOWN_COMMAND, "Unknown lifecycle command", false)
	}
}

func handleHeartbeatMessage(
	conn net.Conn,
	registry *core.Registry,
	comp *core.Component,
	envelope *core.Envelope,
	log *logger.Logger,
) {
	switch envelope.Command {
	case core.CommandPing:
		uptimeSeconds := int64(time.Since(comp.StartedAt).Seconds())
		pongEnvelope, _ := PongResponse(envelope.MessageID, comp.State, uptimeSeconds)
		_ = json.NewEncoder(conn).Encode(pongEnvelope)
		log.Debugf("Sent PONG response")

	case core.CommandPong:
		log.Debugf("Received PONG from component")
		_ = registry.UpdateHeartbeat(comp.ID)

	default:
		log.Warnf("Unknown heartbeat command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, core.UNKNOWN_COMMAND, "Unknown heartbeat command", false)
	}
}

func handleConfigMessage(
	conn net.Conn,
	registry *core.Registry,
	comp *core.Component,
	envelope *core.Envelope,
	log *logger.Logger,
) {
	switch envelope.Command {
	case core.CommandConfigure:
		var payload core.ConfigurePayload
		payloadJSON, _ := json.Marshal(envelope.Payload)
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			log.Errorf("Failed to parse configure payload: %s", err)
			sendErrorResponse(conn, envelope.MessageID, core.INVALID_PAYLOAD, "Failed to parse payload", false)
			return
		}
		if err := registry.UpdateState(comp.ID, core.ComponentStateConfigured); err != nil {
			log.Errorf("Failed to update state to CONFIGURED: %s", err)
			sendErrorResponse(conn, envelope.MessageID, core.STATE_TRANSITION_FAILED, err.Error(), false)
			return
		}
		ackEnvelope, _ := ACKResponse(envelope.MessageID)
		_ = json.NewEncoder(conn).Encode(ackEnvelope)
		log.Debugf("Configuration acknowledged")

	default:
		log.Warnf("Unknown config command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, core.UNKNOWN_COMMAND, "Unknown config command", false)
	}
}

func sendErrorResponse(conn net.Conn, correlationID, code core.ErrorCode, message string, recoverable bool) {
	errorEnvelope, err := ErrorResponse(correlationID, code, message, recoverable)
	if err != nil {
		logger.Errorf("Failed to create error response: %s", err)
		return
	}
	if err := json.NewEncoder(conn).Encode(errorEnvelope); err != nil {
		logger.Errorf("Failed to send error response: %s", err)
	}
}
