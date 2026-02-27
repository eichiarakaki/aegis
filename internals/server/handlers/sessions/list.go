package sessions

import (
	"net"

	"github.com/eichiarakaki/aegis/internals/logger"
)

// HandleSessionList returns all known sessions.
func HandleSessionList(conn net.Conn) {
	logger.Info("Listing sessions")

	// TODO: fetch session records from the store.

	writeJSON(conn, map[string]interface{}{
		"status":   "ok",
		"sessions": []interface{}{},
	})
}
