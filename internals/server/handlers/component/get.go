package component

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	components "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleComponentGet returns component info for a session.
func HandleComponentGet(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Desalinizing payload
	var payload core.ComponentGetPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_GET",
			Status:    "error",
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_GET",
			Status:    "error",
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Getting component %s from session %s", payload.ComponentID, payload.SessionID)

	// Getting the session
	session, found := sessions.GetSessionByHint(payload.SessionID, sessionStore)
	if !found {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_GET",
			Status:    "error",
			Message:   "Session not found",
		})
		return
	}

	// Getting component
	data, err := components.ComponentGet(session, payload.ComponentID)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_GET",
			Status:    "error",
			Message:   fmt.Sprintf("Couldn't get component data: %s", err.Error()),
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "COMPONENT_GET",
		Status:    "ok",
		Data:      data,
	})
}
