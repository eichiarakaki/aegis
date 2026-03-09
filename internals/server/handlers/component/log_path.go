package component

import (
	"encoding/json"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleComponentLogPath returns the filesystem path of the component's log file.
// The CLI uses this to open and tail the file directly — no streaming over the socket.
func HandleComponentLogPath(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	log := logger.WithRequestID(cmd.RequestID)

	var payload core.ComponentLogPathPayload
	raw, _ := json.Marshal(cmd.Payload)
	if err := json.Unmarshal(raw, &payload); err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentLogPath,
			Status:    core.ERROR,
			ErrorCode: core.INVALID_PAYLOAD,
			Message:   err.Error(),
		})
		return
	}

	session, err := sessionStore.GetByHint(payload.SessionID)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentLogPath,
			Status:    core.ERROR,
			ErrorCode: core.SESSION_NOT_FOUND,
			Message:   "session not found: " + payload.SessionID,
		})
		return
	}

	registry := session.Registry
	comp, exists := registry.Get(payload.ComponentID)
	if !exists {
		comp, exists = registry.GetByName(session.ID, payload.ComponentID)
	}
	if !exists {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandComponentLogPath,
			Status:    core.ERROR,
			ErrorCode: core.COMPONENT_NOT_FOUND,
			Message:   "component not found: " + payload.ComponentID,
		})
		return
	}

	logPath := sessions.LogPath(session.ID, comp.ID)
	log.Debugf("Log path for component %s: %s", comp.ID, logPath)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandComponentLogPath,
		Status:    core.OK,
		Data:      logPath,
	})
}
