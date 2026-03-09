package sessions

import (
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

func HandleSessionState(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	payload, err := core.DeserializeSessionActionPayload(cmd)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionDelete,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	session, err := sessions.GetSessionByHint(payload.SessionID, sessionStore)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionState,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionState,
		Status:    core.OK,
		Data:      sessions.GetSessionState(session),
	})
}
