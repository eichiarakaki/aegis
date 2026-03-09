package sessions

import (
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/component"
	services_sessions "github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/utils"
)

// HandleSessionDelete processes SESSION_DELETE commands.
func HandleSessionDelete(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore, connPool *component.ConnectionPool) {
	payload, err := core.DeserializeSessionActionPayload(cmd)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionDelete,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Deleting session: %s", payload.SessionID)

	// Get session
	session, err := sessionStore.GetByHint(payload.SessionID)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionDelete,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	// Store session data before deletion
	sessionID := session.ID
	sessionName := session.Name

	// Delete session
	if err := services_sessions.DeleteSession(session, sessionStore, connPool); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to delete session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionDelete,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to delete session: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session deleted successfully: %s (%s)", sessionName, utils.GetShortHash(sessionID))

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionDelete,
		Status:    core.OK,
		Message:   fmt.Sprintf("Session deleted successfully: %s (%s)", sessionName, utils.GetShortHash(sessionID)),
		Data: map[string]interface{}{
			"session_id":   sessionID,
			"session_name": sessionName,
		},
	})
}
