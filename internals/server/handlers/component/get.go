package component

import (
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
)

func HandleComponentGet(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	var payload core.ComponentGetPayload
	if err := core.DecodePayload(cmd.Payload, &payload); err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentGet,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Invalid payload: %s", err),
		})
		return
	}

	if payload.SessionID == "" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentGet,
			Status:    core.ERROR,
			Message:   "Missing required field: session_id",
		})
		return
	}

	session, err := sessionStore.GetByHint(payload.SessionID)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentGet,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	data, err := servicescomponent.Get(session, payload.ComponentID)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Warnf("Component not found: %s", payload.ComponentID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentGet,
			Status:    core.ERROR,
			Message:   err.Error(),
			Data:      core.ComponentGetData{SessionID: session.ID},
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandComponentGet,
		Status:    core.OK,
		Data:      data,
	})
}
