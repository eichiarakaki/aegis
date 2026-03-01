package sessions

import (
	"fmt"
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

// HandleSessionStart manually starts an existing session by ID.
func HandleSessionStart(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	sessionID := strings.TrimSpace(cmd.Payload)
	// Get session
	session, found := sessions.GetSessionByHint(sessionID, sessionStore)
	if !found {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_START",
			Status:    "error",
			//ErrorCode: "",
			Message: fmt.Sprintf("Session not found."),
			Data:    map[string]interface{}{},
		})

		return
	}

	previousState := session.State

	err := sessions.StartSession(session, sessionStore)
	if err != nil {
		session.State = core.SessionError
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_START",
			Status:    "error",
			//ErrorCode: "",
			Message: fmt.Sprintf("Failed to start session: %s", err.Error()),
			Data: map[string]interface{}{
				"session_id":     session.ID,
				"previous_state": previousState,
				"current_state":  session.State,
				"components":     session.Components,
			},
		})

		return
	}

	session.State = core.SessionStarting

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_START",
		Status:    "ok",
		Data: map[string]interface{}{
			"session_id":     session.ID,
			"previous_state": previousState,
			"current_state":  session.State,
			"started_at":     session.StartedAt,
			"components":     session.Components,
		},
	})
}

// HandleSessionStop manually stops a running session
func HandleSessionStop(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	sessionID := strings.TrimSpace(cmd.Payload)
	session, found := sessions.GetSessionByHint(sessionID, sessionStore)
	if !found {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STOP",
			Status:    "error",
			//ErrorCode: "",
			Message: fmt.Sprintf("Session not found."),
			Data:    map[string]interface{}{},
		})
		return
	}

	previousState := session.State

	err := sessions.StopSession(session, sessionStore)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STOP",
			Status:    "error",
			//ErrorCode: "",
			Message: fmt.Sprintf("Could not stop session: %s", err.Error()),
			Data: map[string]interface{}{
				"session_id": session.ID,
			}})
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "ok",
		Status:    "SESSION_STOP",
		//ErrorCode: "",
		Message: fmt.Sprintf("session %s was stopped successfully.", utils.GetShortHash(session.ID)),
		Data: map[string]interface{}{
			"session_id":     session.ID,
			"previous_state": previousState,
			"current_state":  session.State,
			"stopped_at":     session.StoppedAt,
			"components":     session.Components,
		},
	})
}

// HandleSessionState returns the current status of a session by ID.
func HandleSessionState(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	sessionID := strings.TrimSpace(cmd.Payload)
	session, found := sessions.GetSessionByHint(sessionID, sessionStore)
	if !found {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STATE",
			Status:    "error",
			//ErrorCode: "",
			Message: fmt.Sprintf("Session not found."),
			Data:    map[string]interface{}{},
		})
		return
	}

	data, err := sessions.GetSessionState(cmd, session)

	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STATE",
			Status:    "error",
			//ErrorCode: "",
			Message: fmt.Sprintf("Couldn't get session state."),
			Data:    map[string]interface{}{},
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_STATE",
		Status:    "ok",
		Data:      data,
	})
}
