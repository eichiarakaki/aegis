package sessions

import (
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleSessionList returns all known sessions.
func HandleSessionList(conn net.Conn, sessionStore *core.SessionStore) {
	logger.Debug("Listing sessions")

	allSessions := sessions.ListSessions(sessionStore)
	writeJSON(conn, allSessions)
}
