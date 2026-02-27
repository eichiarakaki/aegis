package sessions

import (
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/logger"
)

// HandleSessionStart manually starts an existing session by ID.
func HandleSessionStart(payload string, conn net.Conn) {
	sessionID := strings.TrimSpace(payload)
	if sessionID == "" {
		writeError(conn, "session_id cannot be empty")
		return
	}

	logger.Infof("Starting session: id=%s", sessionID)

	// TODO: transition session state to running.

	writeJSON(conn, map[string]interface{}{
		"status":     "ok",
		"session_id": sessionID,
		"message":    "session started",
	})
}

// HandleSessionStop manually stops a running session by ID.
func HandleSessionStop(payload string, conn net.Conn) {
	sessionID := strings.TrimSpace(payload)
	if sessionID == "" {
		writeError(conn, "session_id cannot be empty")
		return
	}

	logger.Infof("Stopping session: id=%s", sessionID)

	// TODO: signal all components, close the data-stream socket,
	// and transition session state to stopped.

	writeJSON(conn, map[string]interface{}{
		"status":     "ok",
		"session_id": sessionID,
		"message":    "session stopped",
	})
}
