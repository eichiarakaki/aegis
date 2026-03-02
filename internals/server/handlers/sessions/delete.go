package sessions

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	services_sessions "github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

// HandleSessionDelete processes SESSION_DELETE commands.
func HandleSessionDelete(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.SessionActionPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_DELETE",
			Status:    "error",
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_DELETE",
			Status:    "error",
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required field
	if payload.SessionID == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session deletion failed: missing session_id")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_DELETE",
			Status:    "error",
			Message:   "Missing required field: session_id",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Deleting session: %s", payload.SessionID)

	// Get session
	session, found := services_sessions.GetSessionByHint(payload.SessionID, sessionStore)
	if !found {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_DELETE",
			Status:    "error",
			Message:   "Session not found",
		})
		return
	}

	// Store session data before deletion
	sessionID := session.ID
	sessionName := session.Name

	// Delete session
	if err := services_sessions.DeleteSession(session, sessionStore); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to delete session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_DELETE",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to delete session: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session deleted successfully: %s (%s)", sessionName, utils.GetShortHash(sessionID))

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_DELETE",
		Status:    "ok",
		Message:   fmt.Sprintf("Session deleted successfully: %s (%s)", sessionName, utils.GetShortHash(sessionID)),
		Data: map[string]interface{}{
			"session_id":   sessionID,
			"session_name": sessionName,
		},
	})
}
