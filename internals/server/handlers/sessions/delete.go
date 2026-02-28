package sessions

import (
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleSessionDelete processes SESSION_DELETE commands.
// Payload format: "<session_id>"
func HandleSessionDelete(payload string, conn net.Conn, sessionStore *core.SessionStore) {
	RequestedSessionID := strings.TrimSpace(payload)

	logger.Infof("Deleting session: id=%s", RequestedSessionID)

	id, err := sessions.DeleteSession(RequestedSessionID, sessionStore)
	if err != nil {
		logger.Errorf("Failed to delete session %s: %v", RequestedSessionID, err)
		writeJSON(conn, map[string]any{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	writeJSON(conn, map[string]any{
		"status":     "ok",
		"session_id": id,
		"message":    "session deleted",
	})
}
