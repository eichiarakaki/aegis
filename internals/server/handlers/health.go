package handlers

import (
	"fmt"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	servicehealth "github.com/eichiarakaki/aegis/internals/services/health"
	"github.com/nats-io/nats.go"
)

const heartbeatTimeout = 15 * time.Second

// HandleGlobalHealth returns daemon-level health.
func HandleGlobalHealth(
	cmd core.Command,
	conn net.Conn,
	sessionStore *core.SessionStore,
	nc *nats.Conn,
) {
	data := servicehealth.GlobalHealth(sessionStore, nc)
	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandHealthCheck,
		Status:    core.OK,
		Data:      data,
	})
}

// HandleHealthCheckSession returns session-level health.
func HandleHealthCheckSession(
	cmd core.Command,
	conn net.Conn,
	sessionStore *core.SessionStore,
	pool *servicescomponent.ConnectionPool,
) {
	var payload core.SessionActionPayload
	if err := core.DecodePayload(cmd.Payload, &payload); err != nil || payload.SessionID == "" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandHealthCheckSession,
			Status:    core.ERROR,
			Message:   "Missing required field: session_id",
		})
		return
	}

	session, err := sessionStore.GetByHint(payload.SessionID)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandHealthCheckSession,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	data := servicehealth.SessionHealth(session, pool, heartbeatTimeout)
	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandHealthCheckSession,
		Status:    core.OK,
		Data:      data,
	})
}

// HandleHealthCheckComponent returns component-level health.
func HandleHealthCheckComponent(
	cmd core.Command,
	conn net.Conn,
	sessionStore *core.SessionStore,
	pool *servicescomponent.ConnectionPool,
) {
	var payload core.ComponentGetPayload
	if err := core.DecodePayload(cmd.Payload, &payload); err != nil || payload.SessionID == "" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandHealthCheckComp,
			Status:    core.ERROR,
			Message:   "Missing required fields: session_id, component_id",
		})
		return
	}

	session, err := sessionStore.GetByHint(payload.SessionID)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandHealthCheckComp,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	comp, err := resolveComponent(session, payload.ComponentID)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandHealthCheckComp,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("component %q not found: %s", payload.ComponentID, err),
		})
		return
	}

	data := servicehealth.ComponentHealth(session, comp, pool, heartbeatTimeout)
	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandHealthCheckComp,
		Status:    core.OK,
		Data:      data,
	})
}

// resolveComponent finds a component by exact ID or name (delegates to
// the component service resolver to avoid duplicating logic).
func resolveComponent(session *core.Session, ref string) (*core.Component, error) {
	if ref == "" {
		all := session.Registry.List()
		if len(all) == 1 {
			return all[0], nil
		}
		return nil, fmt.Errorf("specify a component_id")
	}
	if c, ok := session.Registry.Get(ref); ok {
		return c, nil
	}
	if c, ok := session.Registry.GetByName(session.ID, ref); ok {
		return c, nil
	}
	return nil, fmt.Errorf("not found")
}
