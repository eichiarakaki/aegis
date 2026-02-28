package sessions

import (
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleSessionStart manually starts an existing session by ID.
func HandleSessionStart(payload string, conn net.Conn, sessionStore *core.SessionStore) {
	sessionID := strings.TrimSpace(payload)

	err := sessions.StartSession(sessionID, sessionStore)
	if err != nil {
		writeJSON(conn, map[string]any{
			"status":     "error",
			"session_id": sessionID,
			"message":    err.Error(),
		})

		return
	}

	writeJSON(conn, map[string]any{
		"status":     "ok",
		"session_id": sessionID,
		"message":    "session started",
	})
}

// HandleSessionStop manually stops a running session by ID.
func HandleSessionStop(payload string, conn net.Conn, sessionStore *core.SessionStore) {
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

// HandleSessionStatus returns the current status of a session by ID.
func HandleSessionStatus(conn net.Conn, payload string, sessionStore *core.SessionStore) {
	sessionID := strings.TrimSpace(payload)
	if sessionID == "" {
		writeError(conn, "session_id cannot be empty")
		return
	}

	logger.Infof("Getting status for session: id=%s", sessionID)

	// For now, we return a mock response.
	writeJSON(conn, map[string]interface{}{
		"status":     "ok",
		"session_id": sessionID,
		"state":      "running",
		"started_at": "2024-01-01T12:00:00Z",
	})
}
