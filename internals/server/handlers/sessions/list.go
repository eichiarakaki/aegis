package sessions

import (
	"net"

	"github.com/eichiarakaki/aegis/internals/logger"
)

// HandleSessionList returns all known sessions.
func HandleSessionList(conn net.Conn) {
	logger.Info("Listing sessions")

	// For now, we return a mock response.
	writeJSON(conn, map[string]any{
		"status": "ok",
		"sessions": []any{
			map[string]any{
				"id":         "session-123",
				"name":       "My Session",
				"mode":       "live",
				"state":      "running",
				"started_at": "2024-01-01T12:00:00Z",
			},
			map[string]any{
				"id":         "session-456",
				"name":       "Backtest Session",
				"mode":       "backtest",
				"state":      "stopped",
				"started_at": "2024-01-02T08:30:00Z",
			},
		},
	})
}
