package component

import (
	"fmt"
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	components "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleComponentList returns all known components of a session.
func HandleComponentList(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	sessionID := strings.TrimSpace(cmd.Payload)
	session, found := sessions.GetSessionByHint(sessionID, sessionStore)
	if !found {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_LIST",
			Status:    "error",
			// ErrorCode: "",
			Message: "Session not found.",
			Data:    nil,
		})
		return
	}

	data, err := components.ComponentList(session)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_LIST",
			Status:    "error",
			// ErrorCode: "",
			Message: fmt.Sprintf("Couldn't get the component data: %s", err.Error()),
			Data:    nil,
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "COMPONENT_LIST",
		Status:    "ok",
		//ErrorCode: "",
		//Message:   "",
		Data: data,
	})
}
