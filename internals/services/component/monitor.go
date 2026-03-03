package component

import (
	"encoding/json"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// ComponentHeartbeatMonitor monitors component health across all sessions
// and sends periodic PING messages to detect dead components.
type ComponentHeartbeatMonitor struct {
	sessionStore *core.SessionStore
	pool         *ConnectionPool
	interval     time.Duration
	timeout      time.Duration
}

// NewComponentHeartbeatMonitor creates a new heartbeat monitor.
func NewComponentHeartbeatMonitor(
	sessionStore *core.SessionStore,
	pool *ConnectionPool,
) *ComponentHeartbeatMonitor {
	return &ComponentHeartbeatMonitor{
		sessionStore: sessionStore,
		pool:         pool,
		interval:     5 * time.Second,
		timeout:      15 * time.Second,
	}
}

// Start begins monitoring component health on a fixed interval.
func (m *ComponentHeartbeatMonitor) Start() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for range ticker.C {
		m.checkComponents()
	}
}

// checkComponents iterates over every session and its registry,
// sending PINGs to live components and cleaning up dead ones.
func (m *ComponentHeartbeatMonitor) checkComponents() {
	for _, session := range m.sessionStore.ListSessions() {
		if session.Registry == nil {
			continue
		}

		for _, comp := range session.Registry.List() {
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

// handleDeadComponent transitions the component to ERROR state, notifies its
// parent session, closes its connection, and unregisters it from the registry.
func (m *ComponentHeartbeatMonitor) handleDeadComponent(
	registry *component.ComponentRegistry,
	comp *component.Component,
	log *logger.Logger,
) {
	// 1. Transition component state to ERROR
	if err := registry.UpdateState(comp.ID, component.ComponentStateError); err != nil {
		log.Errorf("Failed to transition component to ERROR state: %s", err.Error())
	}

	// 2. Notify the parent session so it can stop gracefully
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

	// 4. Unregister from the session's registry
	if err := registry.Unregister(comp.ID); err != nil {
		log.Errorf("Failed to unregister dead component: %s", err.Error())
	}

	log.Infof("Dead component fully cleaned up")
}
