package component

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	services_component "github.com/eichiarakaki/aegis/internals/services/component"
	servicessessions "github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleComponentList returns all components in a session.
func HandleComponentList(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.ComponentListPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_LIST",
			Status:    "error",
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_LIST",
			Status:    "error",
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required field
	if payload.SessionID == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Component list failed: missing session_id")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_LIST",
			Status:    "error",
			Message:   "Missing required field: session_id",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Debugf("Listing components for session: %s", payload.SessionID)

	// Get session
	session, found := servicessessions.GetSessionByHint(payload.SessionID, sessionStore)
	if !found {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_LIST",
			Status:    "error",
			Message:   "Session not found",
		})
		return
	}

	// List components
	data, err := services_component.ComponentList(session)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to list components: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "COMPONENT_LIST",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to list components: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Debugf("Components listed successfully for session: %s", session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "COMPONENT_LIST",
		Status:    "ok",
		Data:      data,
	})
}
