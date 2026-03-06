package sessions

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/utils"
	"github.com/nats-io/nats.go"
)

// HandleSessionStart starts an existing session.
func HandleSessionStart(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore, nc *nats.Conn, logStore *servicescomponent.LogStore) {
	// Deserialize payload
	var payload core.SessionActionPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStart,
			Status:    core.ERROR,
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStart,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required field
	if payload.SessionID == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session start failed: missing session_id")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStart,
			Status:    core.ERROR,
			Message:   "Missing required field: session_id",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Starting session: %s", payload.SessionID)

	// Get session
	session, err := sessions.GetSessionByHint(payload.SessionID, sessionStore)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Warnf("Session not found: %s", payload.SessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStart,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	previousState := session.State

	// Start session
	if err := sessions.StartSession(session, cmd, conn, nc); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to start session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_START",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to start session: %s", err.Error()),
			Data: map[string]any{
				"session_id":     session.ID,
				"previous_state": string(previousState),
				"current_state":  string(core.SessionError),
			},
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session started successfully: %s", session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionStart,
		Status:    core.OK,
		Message:   fmt.Sprintf("Session started successfully: %s", utils.GetShortHash(session.ID)),
		Data: map[string]any{
			"session_id":     session.ID,
			"previous_state": string(previousState),
			"current_state":  string(session.State),
			"started_at":     session.StartedAt,
			"components":     session.Registry.List(),
		},
	})
}
