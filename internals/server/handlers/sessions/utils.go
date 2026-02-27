package sessions

import (
	"encoding/json"
	"net"

	"github.com/eichiarakaki/aegis/internals/logger"
)

// writeJSON sends a JSON-encoded response through the connection.
func writeJSON(conn net.Conn, v any) {
	if err := json.NewEncoder(conn).Encode(v); err != nil {
		logger.Warnf("Failed to write response: %v", err)
	}
}

// writeError sends a JSON error response through the connection.
func writeError(conn net.Conn, msg string) {
	writeJSON(conn, map[string]interface{}{
		"status": "error",
		"error":  msg,
	})
}
