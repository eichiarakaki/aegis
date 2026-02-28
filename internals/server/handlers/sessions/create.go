package sessions

import (
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	service_sessions "github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleSessionCreate processes SESSION_CREATE commands.
// Payload format: "<name>|<mode>"
func HandleSessionCreate(payload string, conn net.Conn, sessionStore *core.SessionStore) {
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) != 2 {
		writeError(conn, "invalid payload: expected <name>|<mode>")
		return
	}

	name, mode := parts[0], parts[1]

	if name == "" {
		writeError(conn, "session name cannot be empty")
		return
	}

	if mode != "realtime" && mode != "historical" {
		writeError(conn, "invalid mode: must be 'realtime' or 'historical'")
		return
	}

	// Creating the session
	token, err := service_sessions.CreateSession(name, mode, sessionStore)
	if err != nil {
		writeJSON(conn, map[string]any{
			"status":       "error",
			"session_name": name,
			"message":      "failed to create session",
			"error":        err.Error(),
		})
		return
	}

	writeJSON(conn, map[string]any{
		"status":       "ok",
		"session_name": name,
		"session_id":   token,
		"mode":         mode,
	})
}

// HandleSessionCreateRun creates a new session and immediately spawns
// the provided components under a fresh SessionToken.
//
// Payload: <n>|<mode>|<path1>,<path2>,...
func HandleSessionCreateRun(payload string, conn net.Conn, sessionStore *core.SessionStore) {
	name, mode, paths, err := parseRunPayload(payload)
	if err != nil {
		writeError(conn, err.Error())
		return
	}

	if mode != "realtime" && mode != "historical" {
		writeError(conn, "invalid mode: must be 'realtime' or 'historical'")
		return
	}

	logger.Infof("Creating session and running components: name=%s mode=%s paths=%v", name, mode, paths)

	// TODO:
	//   1. Persist the new session record and generate a SessionToken.
	//   2. For each path, exec the binary with AEGIS_SESSION_TOKEN=<token>.
	//   3. Components connect to /tmp/aegis-data-stream-<session_id>.sock
	//      once the token is verified.

	writeJSON(conn, map[string]interface{}{
		"status":     "ok",
		"session":    name,
		"mode":       mode,
		"components": paths,
	})
}
