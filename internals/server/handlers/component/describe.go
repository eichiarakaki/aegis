package component

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	servicessessions "github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleComponentDescribe returns detailed information about all components in a session.
func HandleComponentDescribe(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.ComponentListPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_DESCRIBE",
			Status:    "error",
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_DESCRIBE",
			Status:    "error",
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required field
	if payload.SessionID == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Component describe failed: missing session_id")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_DESCRIBE",
			Status:    "error",
			Message:   "Missing required field: session_id",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Debugf("Describing components for session: %s", payload.SessionID)

	// Get session
	session, found := servicessessions.GetSessionByHint(payload.SessionID, sessionStore)
	if !found {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_DESCRIBE",
			Status:    "error",
			Message:   "Session not found",
		})
		return
	}

	// Describe components
	data, err := servicescomponent.ComponentDescribe(session)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to describe components: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_DESCRIBE",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to describe components: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Debugf("Components described successfully for session: %s", session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "COMPONENT_DESCRIBE",
		Status:    "ok",
		Data:      data,
	})
}
