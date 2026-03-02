package sessions

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicessessions "github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

// HandleSessionStart starts an existing session.
func HandleSessionStart(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.SessionActionPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_START",
			Status:    "error",
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_START",
			Status:    "error",
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required field
	if payload.SessionID == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session start failed: missing session_id")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_START",
			Status:    "error",
			Message:   "Missing required field: session_id",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Starting session: %s", payload.SessionID)

	// Get session
	session, found := servicessessions.GetSessionByHint(payload.SessionID, sessionStore)
	if !found {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_START",
			Status:    "error",
			Message:   "Session not found",
		})
		return
	}

	previousState := session.State

	// Start session
	if err := servicessessions.StartSession(session, sessionStore); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to start session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_START",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to start session: %s", err.Error()),
			Data: map[string]interface{}{
				"session_id":     session.ID,
				"previous_state": core.SessionStateToString(previousState),
				"current_state":  core.SessionStateToString(core.SessionError),
			},
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session started successfully: %s", session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_START",
		Status:    "ok",
		Message:   fmt.Sprintf("Session started successfully: %s", utils.GetShortHash(session.ID)),
		Data: map[string]interface{}{
			"session_id":     session.ID,
			"previous_state": core.SessionStateToString(previousState),
			"current_state":  core.SessionStateToString(session.State),
			"started_at":     session.StartedAt,
			"components":     session.Components,
		},
	})
}

// HandleSessionStop stops a running session.
func HandleSessionStop(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.SessionActionPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STOP",
			Status:    "error",
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STOP",
			Status:    "error",
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required field
	if payload.SessionID == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session stop failed: missing session_id")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STOP",
			Status:    "error",
			Message:   "Missing required field: session_id",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Stopping session: %s", payload.SessionID)

	// Get session
	session, found := servicessessions.GetSessionByHint(payload.SessionID, sessionStore)
	if !found {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STOP",
			Status:    "error",
			Message:   "Session not found",
		})
		return
	}

	previousState := session.State

	// Stop session
	if err := servicessessions.StopSession(session, sessionStore); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to stop session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STOP",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to stop session: %s", err.Error()),
			Data: map[string]interface{}{
				"session_id": session.ID,
			},
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session stopped successfully: %s", session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_STOP",
		Status:    "ok",
		Message:   fmt.Sprintf("Session stopped successfully: %s", utils.GetShortHash(session.ID)),
		Data: map[string]interface{}{
			"session_id":     session.ID,
			"previous_state": core.SessionStateToString(previousState),
			"current_state":  core.SessionStateToString(session.State),
			"stopped_at":     session.StoppedAt,
			"components":     session.Components,
		},
	})
}

// HandleSessionState returns the current state of a session.
func HandleSessionState(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.SessionActionPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STATE",
			Status:    "error",
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STATE",
			Status:    "error",
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required field
	if payload.SessionID == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session state query failed: missing session_id")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STATE",
			Status:    "error",
			Message:   "Missing required field: session_id",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Debugf("Querying session state: %s", payload.SessionID)

	// Get session
	session, found := servicessessions.GetSessionByHint(payload.SessionID, sessionStore)
	if !found {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STATE",
			Status:    "error",
			Message:   "Session not found",
		})
		return
	}

	// Get session state
	data, err := servicessessions.GetSessionState(cmd, session)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to retrieve session state: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_STATE",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to retrieve session state: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Debugf("Session state retrieved successfully: %s", session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_STATE",
		Status:    "ok",
		Data:      data,
	})
}
