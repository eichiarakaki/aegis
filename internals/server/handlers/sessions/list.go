package sessions

import (
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

func HandleSessionList(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	if sessionStore.Count() == 0 {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   string(core.CommandSessionList),
			Status:    "ok",
			//ErrorCode: "",
			Message: "There are 0 sessions at the moment",
			Data:    nil,
		})
	}
	data := sessions.ListSessions(sessionStore)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   string(core.CommandComponentList),
		Status:    "ok",
		//ErrorCode: "",
		//Message:   "",
		Data: data,
	})
}
