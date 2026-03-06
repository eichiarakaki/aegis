package component

import (
	"encoding/json"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// HeartbeatMonitor monitors component health across all sessions
// and sends periodic PING messages to detect dead components.
type HeartbeatMonitor struct {
	sessionStore *core.SessionStore
	pool         *ConnectionPool
	interval     time.Duration
	timeout      time.Duration
}

// NewComponentHeartbeatMonitor creates a new heartbeat monitor.
func NewComponentHeartbeatMonitor(
	sessionStore *core.SessionStore,
	pool *ConnectionPool,
) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		sessionStore: sessionStore,
		pool:         pool,
		interval:     5 * time.Second,
		timeout:      15 * time.Second,
	}
}

// Start begins monitoring component health on a fixed interval.
func (m *HeartbeatMonitor) Start() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for range ticker.C {
		m.checkComponents()
	}
}

// checkComponents iterates over every session and its registry,
// sending PINGs to live components and cleaning up dead ones.
func (m *HeartbeatMonitor) checkComponents() {
	for _, session := range m.sessionStore.ListSessions() {
		if session.Registry == nil {
			continue
		}

		for _, comp := range session.Registry.List() {
			// Skip components that haven't completed the handshake yet.
			// INIT      = placeholder registered during attach, process not started.
			// REGISTERED = process connected but still in WaitForReady.
			// Heartbeating these would always fail and cause spurious cleanup.
			if comp.State == core.ComponentStateInit ||
				comp.State == core.ComponentStateRegistered ||
				comp.State == core.ComponentStateInitializing {
				continue
			}

			log := logger.WithComponent("HeartbeatMonitor").
				WithField("session_id", session.ID).
				WithField("component_id", comp.ID).
				WithField("component_name", comp.Name)

			timeSinceLastHeartbeat := time.Since(comp.LastHeartbeat)

			if timeSinceLastHeartbeat > m.timeout {
				log.Warnf("Component timed out — last heartbeat was %.0fs ago", timeSinceLastHeartbeat.Seconds())
				m.handleDeadComponent(session.Registry, comp, log)
				continue
			}

			m.sendPing(comp, log)
		}
	}
}

// sendPing sends a PING message to the component through its active connection.
func (m *HeartbeatMonitor) sendPing(comp *core.Component, log *logger.Logger) {
	conn, exists := m.pool.Get(comp.ID)
	if !exists {
		log.Warnf("No active connection found for component, skipping PING")
		return
	}

	pingEnvelope := core.NewEnvelope(
		core.MessageTypeHeartbeat,
		core.CommandPing,
		"aegis",
		"component:"+comp.Name,
		map[string]any{},
	)

	if err := json.NewEncoder(conn).Encode(pingEnvelope); err != nil {
		log.Warnf("Failed to send PING: %s", err.Error())
	}
}

// handleDeadComponent transitions the component to ERROR state, notifies its
// parent session, closes its connection, and unregisters it from the registry.
func (m *HeartbeatMonitor) handleDeadComponent(
	registry *core.Registry,
	comp *core.Component,
	log *logger.Logger,
) {
	// 1. Transition component state to ERROR
	if err := registry.UpdateState(comp.ID, core.ComponentStateError); err != nil {
		log.Errorf("Failed to transition component to ERROR state: %s", err.Error())
	}

	// 2. Notify the parent session so it can stop gracefully
	// if session, exists := m.sessionStore.GetSessionByID(comp.SessionID); exists {
	// 	if err := session.SetToStopping(); err != nil {
	// 		log.Warnf("Failed to stop parent session %s: %s", session.ID, err.Error())
	// 	} else {
	// 		log.Warnf("Parent session %s transitioned to STOPPED due to dead component", session.ID)
	// 	}
	// }

	// 3. Close the active connection
	if conn, exists := m.pool.Get(comp.ID); exists {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection for dead component: %s", err.Error())
		}
		m.pool.Remove(comp.ID)
	}

	// 4. Unregister from the session's registry
	if err := registry.Unregister(comp.ID); err != nil {
		log.Errorf("Failed to unregister dead component: %s", err.Error())
	}

	log.Infof("Dead component fully cleaned up")
}
