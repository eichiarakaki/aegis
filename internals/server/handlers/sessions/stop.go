package sessions

import (
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/utils"
)

// HandleSessionStop stops a running session.
func HandleSessionStop(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	payload, err := core.DeserializeSessionActionPayload(cmd)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStop,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	// Validate required field
	if payload.SessionID == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session stop failed: missing session_id")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStop,
			Status:    core.ERROR,
			Message:   "Missing required field: session_id",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Stopping session: %s", payload.SessionID)

	// Get session
	session, err := sessions.GetSessionByHint(payload.SessionID, sessionStore)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStop,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	previousState := session.State

	// Stop session
	if err := sessions.StopSession(session, sessionStore); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to stop session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStop,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to stop session: %s", err.Error()),
			Data: map[string]any{
				"session_id": session.ID,
			},
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session stopped successfully: %s", session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionStop,
		Status:    core.OK,
		Message:   fmt.Sprintf("Session stopped successfully: %s", utils.GetShortHash(session.ID)),
		Data: map[string]any{
			"session_id":     session.ID,
			"previous_state": string(previousState),
			"current_state":  string(session.State),
			"stopped_at":     session.StoppedAt,
			"components":     session.Registry.List(),
		},
	})
}
