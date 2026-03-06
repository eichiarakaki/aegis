package sessions

import (
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

func HandleSessionList(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	data := sessions.ListSessions(sessionStore)
	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionList,
		Status:    core.OK,
		Data:      data,
	})
}
