package sessions

import (
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/logger"
)

// HandleSessionDelete processes SESSION_DELETE commands.
// Payload format: "<session_id>"
func HandleSessionDelete(payload string, conn net.Conn) {
	sessionID := strings.TrimSpace(payload)

	if sessionID == "" {
		writeError(conn, "session_id cannot be empty")
		return
	}

	logger.Infof("Deleting session: id=%s", sessionID)

	// TODO: look up session by ID, teardown resources, and remove from store.

	writeJSON(conn, map[string]interface{}{
		"status":     "ok",
		"session_id": sessionID,
		"message":    "session deleted",
	})
}
