package component

import (
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	servicessessions "github.com/eichiarakaki/aegis/internals/services/sessions"
)

func HandleComponentDescribe(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	var payload core.ComponentGetPayload
	if err := core.DecodePayload(cmd.Payload, &payload); err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentDescribe,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Invalid payload: %s", err),
		})
		return
	}

	if payload.SessionID == "" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentDescribe,
			Status:    core.ERROR,
			Message:   "Missing required field: session_id",
		})
		return
	}

	session, err := servicessessions.GetSessionByHint(payload.SessionID, sessionStore)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentDescribe,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	data, err := servicescomponent.Describe(session, payload.ComponentID)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Warnf("Component not found: %s", payload.ComponentID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentDescribe,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandComponentDescribe,
		Status:    core.OK,
		Data:      data,
	})
}
