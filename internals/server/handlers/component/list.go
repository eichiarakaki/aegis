package component

import (
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
)

func HandleComponentList(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	var payload core.ComponentListPayload
	if err := core.DecodePayload(cmd.Payload, &payload); err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentList,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Invalid payload: %s", err),
		})
		return
	}

	if payload.SessionID == "" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentList,
			Status:    core.ERROR,
			Message:   "Missing required field: session_id",
		})
		return
	}

	session, err := sessionStore.GetByHint(payload.SessionID)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentList,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandComponentList,
		Status:    core.OK,
		Data:      servicescomponent.List(session),
	})
}
