package sessions

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	sessionsvc "github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

// HandleSessionAttach attaches new components to an existing session.
func HandleSessionAttach(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.SessionAttachPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required fields
	if payload.SessionID == "" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			Message:   "Missing required field: session_id",
		})
		return
	}

	if len(payload.Paths) == 0 {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			Message:   "At least one component path is required",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Attaching %d components to session %s", len(payload.Paths), payload.SessionID)

	// Get session
	session, err := sessionsvc.GetSessionByHint(payload.SessionID, sessionStore)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			Message:   err.Error(),
			Data: map[string]string{
				"session_id": payload.SessionID,
			},
		})
		return
	}

	// Attach components
	components, err := sessionsvc.AttachComponents(session, payload.Paths)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to attach components: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to attach components: %s", err.Error()),
			Data: map[string]string{
				"session_id": session.ID,
			},
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Successfully attached %d components to session %s", len(components), session.Name)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_ATTACH",
		Status:    "ok",
		Message:   fmt.Sprintf("Attached %d components to %s (%s)", len(components), session.Name, utils.GetShortHash(session.ID)),
		Data: map[string]interface{}{
			"session_id":          session.ID,
			"attached_components": components,
		},
	})
}
