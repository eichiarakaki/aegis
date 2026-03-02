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

// HandleComponentConnection ComponentConnectionHandler manages incoming connections from components.
// It handles the registration handshake and lifecycle management.
func HandleComponentConnection(conn net.Conn, registry *component.ComponentRegistry, sessionStore *core.SessionStore, pool *ConnectionPool) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
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

	// Validating session token against the stored session tokens
	session, err := sessions.GetSessionByHint(registerPayload.SessionToken, sessionStore)
	if err != nil {
		logging.Error("The Token provided doesn't match with any session token.")
		sendErrorResponse(conn, registerEnvelope.MessageID, "WRONG_SESSION_TOKEN", "The Token provided doesn't match with any session token.", false)
		return
	}

	// STEP 2: Create component record
	componentID := "cmp-" + uuid.New().String()[:8]
	comp := &component.Component{
		ID:            componentID,
		Name:          registerPayload.ComponentName,
		Version:       registerPayload.Version,
		State:         component.ComponentStateRegistered,
		Capabilities:  registerPayload.Capabilities,
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
	}

	// Register component
	if err := registry.Register(comp); err != nil {
		logging.Errorf("Failed to register component: %s", err.Error())
		sendErrorResponse(conn, registerEnvelope.MessageID, "REGISTRATION_FAILED", err.Error(), false)
		return
	}

	logging = logging.WithField("component_id", componentID)
	logging.Infof("Component registered successfully")

	// Log capabilities
	logging.Debugf("Supported symbols: %v", registerPayload.Capabilities.SupportedSymbols)
	logging.Debugf("Supported timeframes: %v", registerPayload.Capabilities.SupportedTimeframes)
	logging.Debugf("Requires streams: %v", registerPayload.Capabilities.RequiresStreams)

	// STEP 3: Send REGISTERED response
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
	defer pool.Remove(componentID) // cleanup

	// STEP 4: Handle component lifecycle messages
	handleComponentLifecycle(conn, registry, comp, logging)
}

// handleComponentLifecycle manages the component's lifecycle and heartbeats.
func handleComponentLifecycle(
	conn net.Conn,
	registry *component.ComponentRegistry,
	comp *component.Component,
	logger *logger.Logger,
) {
	// Set read timeout for heartbeat detection
	err := conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		return
	}

	for {
		var envelope component.Envelope
		if err := json.NewDecoder(conn).Decode(&envelope); err != nil {
			logger.Warnf("Connection closed or error reading message: %s", err.Error())
			_ = registry.Unregister(comp.ID)
			logger.Infof("Component unregistered")
			return
		}

		// Reset read deadline after successful read
		err := conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		if err != nil {
			return
		}

		// Validate envelope
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

		// Update component state
		if err := registry.UpdateState(comp.ID, payload.State); err != nil {
			logger.Warnf("Failed to update state: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, "STATE_TRANSITION_FAILED", err.Error(), false)
			return
		}

		logger.Infof("Component state updated: %s", payload.State)

		// Send ACK
		ackEnvelope, _ := component.ACKResponse(envelope.MessageID)
		err := json.NewEncoder(conn).Encode(ackEnvelope)
		if err != nil {
			return
		}

	case component.CommandShutdown:
		logger.Infof("Component initiated shutdown")
		ackEnvelope, _ := component.ACKResponse(envelope.MessageID)
		err := json.NewEncoder(conn).Encode(ackEnvelope)
		if err != nil {
			return
		}
		err = registry.Unregister(comp.ID)
		if err != nil {
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
		// Aegis sends PING, component responds with PONG
		uptimeSeconds := int64(time.Since(comp.StartedAt).Seconds())
		pongEnvelope, _ := component.PongResponse(envelope.MessageID, comp.State, uptimeSeconds)
		err := json.NewEncoder(conn).Encode(pongEnvelope)
		if err != nil {
			return
		}
		logger.Debugf("Sent PONG response")

	case component.CommandPong:
		// Component responds to our PING
		logger.Debugf("Received PONG from component")
		err := registry.UpdateHeartbeat(comp.ID)
		if err != nil {
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

		// Update state to CONFIGURED
		if err := registry.UpdateState(comp.ID, component.ComponentStateConfigured); err != nil {
			logger.Errorf("Failed to update state to CONFIGURED: %s", err.Error())
			sendErrorResponse(conn, envelope.MessageID, "STATE_TRANSITION_FAILED", err.Error(), false)
			return
		}

		// Send ACK
		ackEnvelope, _ := component.ACKResponse(envelope.MessageID)
		err := json.NewEncoder(conn).Encode(ackEnvelope)
		if err != nil {
			return
		}
		logger.Debugf("Configuration acknowledged")

	default:
		logger.Warnf("Unknown config command: %s", envelope.Command)
		sendErrorResponse(conn, envelope.MessageID, "UNKNOWN_COMMAND", "Unknown config command", false)
	}
}

// sendErrorResponse sends an error response envelope.
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

// ComponentHeartbeatMonitor monitors component health and sends periodic heartbeats.
type ComponentHeartbeatMonitor struct {
	registry     *component.ComponentRegistry
	sessionStore *core.SessionStore
	pool         *ConnectionPool
	interval     time.Duration
	timeout      time.Duration
}

// NewComponentHeartbeatMonitor creates a new heartbeat monitor.
func NewComponentHeartbeatMonitor(
	registry *component.ComponentRegistry,
	sessionStore *core.SessionStore,
	pool *ConnectionPool,
) *ComponentHeartbeatMonitor {
	return &ComponentHeartbeatMonitor{
		registry:     registry,
		sessionStore: sessionStore,
		pool:         pool,
		interval:     5 * time.Second,
		timeout:      15 * time.Second,
	}
}

// Start begins monitoring component health.
func (m *ComponentHeartbeatMonitor) Start() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for range ticker.C {
		m.checkComponents()
	}
}

// checkComponents sends a PING to every registered component and unregisters
// those that have exceeded the heartbeat timeout.
func (m *ComponentHeartbeatMonitor) checkComponents() {
	components := m.registry.List()

	for _, comp := range components {
		log := logger.WithComponent("HeartbeatMonitor").
			WithField("component_id", comp.ID).
			WithField("component_name", comp.Name)

		timeSinceLastHeartbeat := time.Since(comp.LastHeartbeat)

		if timeSinceLastHeartbeat > m.timeout {
			log.Warnf("Component timed out — last heartbeat was %.0fs ago", timeSinceLastHeartbeat.Seconds())
			m.handleDeadComponent(comp, log)
			continue
		}

		// Component is alive, send PING
		m.sendPing(comp, log)
	}
}

// sendPing sends a PING message to the component through its active connection.
func (m *ComponentHeartbeatMonitor) sendPing(comp *component.Component, log *logger.Logger) {
	conn, exists := m.pool.Get(comp.ID)
	if !exists {
		log.Warnf("No active connection found for component, skipping PING")
		return
	}

	pingEnvelope := component.NewEnvelope(
		component.MessageTypeHeartbeat,
		component.CommandPing,
		"aegis",
		"component:"+comp.Name,
		map[string]any{},
	)

	if err := json.NewEncoder(conn).Encode(pingEnvelope); err != nil {
		log.Warnf("Failed to send PING: %s", err.Error())
	}
}

// handleDeadComponent transitions the component to ERROR, notifies its parent session,
// closes its connection, and unregisters it from the registry.
func (m *ComponentHeartbeatMonitor) handleDeadComponent(comp *component.Component, log *logger.Logger) {
	// 1. Transition component state to ERROR
	if err := m.registry.UpdateState(comp.ID, component.ComponentStateError); err != nil {
		log.Errorf("Failed to transition component to ERROR state: %s", err.Error())
	}

	// 2. Notify parent session
	if session, exists := m.sessionStore.GetSessionByID(comp.SessionID); exists {
		if err := session.SetToStop(); err != nil {
			log.Warnf("Failed to stop parent session %s: %s", session.ID, err.Error())
		} else {
			log.Warnf("Parent session %s transitioned to STOPPED due to dead component", session.ID)
		}
	}

	// 3. Close the active connection
	if conn, exists := m.pool.Get(comp.ID); exists {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection for dead component: %s", err.Error())
		}
		m.pool.Remove(comp.ID)
	}

	// 4. Unregister from registry
	if err := m.registry.Unregister(comp.ID); err != nil {
		log.Errorf("Failed to unregister dead component: %s", err.Error())
	}

	log.Infof("Dead component fully cleaned up")
}
