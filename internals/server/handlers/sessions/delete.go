package sessions

import (
	"fmt"
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

// HandleSessionDelete processes SESSION_DELETE commands.
// Payload format: "<session_id>"
func HandleSessionDelete(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	hint := strings.TrimSpace(cmd.Payload)
	session, found := sessions.GetSessionByHint(hint, sessionStore)
	if !found {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_DELETE",
			Status:    "error",
			//ErrorCode: "",
			Message: fmt.Sprintf("Session not found."),
			Data:    map[string]interface{}{},
		})
		return
	}

	// Getting a copy of session's data as it is going to be unavailable
	sessionID := session.ID
	sessionName := session.Name

	err := sessions.DeleteSession(session, sessionStore)

	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_DELETE",
			Status:    "error",
			//ErrorCode: "",
			Message: fmt.Sprintf("Couldn't remove session: %s", err.Error()),
			Data:    map[string]interface{}{},
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_DELETE",
		Status:    "ok",
		//ErrorCode: "",
		Message: fmt.Sprintf("%s (%s) was deleted successfully", sessionName, utils.GetShortHash(sessionID)),
		Data: map[string]interface{}{
			"session_id":   sessionID,
			"session_name": sessionName,
		},
	})
}
