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
			// Skip all states that are part of the handshake sequence or that
			// haven't started communicating yet. Only RUNNING and WAITING are
			// steady-state and should be heartbeat-monitored.
			//
			// FIX: previously only INIT/REGISTERED/INITIALIZING were skipped.
			// READY and CONFIGURED are also transient handshake states — the
			// component reaches them before it enters the steady-state lifecycle
			// loop. More importantly, LastHeartbeat is zero-valued at creation
			// so time.Since(LastHeartbeat) >> timeout and the monitor killed the
			// component immediately after it finished the handshake.
			switch comp.State {
			case core.ComponentStateInit,
				core.ComponentStateRegistered,
				core.ComponentStateInitializing,
				core.ComponentStateReady,
				core.ComponentStateConfigured:
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
	if err := registry.UpdateState(comp.ID, core.ComponentStateError); err != nil {
		log.Errorf("Failed to transition component to ERROR state: %s", err.Error())
	}

	if conn, exists := m.pool.Get(comp.ID); exists {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection for dead component: %s", err.Error())
		}
		m.pool.Remove(comp.ID)
	}

	if err := registry.Unregister(comp.ID); err != nil {
		log.Errorf("Failed to unregister dead component: %s", err.Error())
	}

	// Ignore NOT_FOUND — handleComponentLifecycle may have already unregistered
	// the component when it detected the connection drop concurrently.
	if err := registry.Unregister(comp.ID); err != nil && !core.IsNotFound(err) {
		log.Errorf("Failed to unregister dead component: %s", err.Error())
	}

	log.Infof("Dead component fully cleaned up")
}
