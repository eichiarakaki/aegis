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

// HandleSessionRestart restarts a FINISHED session without relaunching component processes.
func HandleSessionRestart(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore, nc *nats.Conn, logStore *servicescomponent.LogStore) {
	var payload core.SessionStartPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionRestart,
			Status:    core.ERROR,
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionRestart,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	if payload.SessionID == "" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionRestart,
			Status:    core.ERROR,
			Message:   "Missing required field: session_id",
		})
		return
	}

	session, err := sessions.GetSessionByHint(payload.SessionID, sessionStore)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionRestart,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	if session.GetState() != core.SessionFinished {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionRestart,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("restart is only valid for FINISHED sessions (current state: %s)", session.GetState()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Restarting session: %s (from=%d to=%d)", payload.SessionID, payload.From, payload.To)

	tr := sessions.TimeRange{From: payload.From, To: payload.To}
	if err := sessions.RestartSession(session, cmd, conn, nc, tr); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to restart session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionRestart,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to restart session: %s", err.Error()),
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionRestart,
		Status:    core.OK,
		Message:   fmt.Sprintf("Session restarted: %s", utils.GetShortHash(session.ID)),
		Data: map[string]any{
			"session_id":    session.ID,
			"current_state": string(session.State),
			"started_at":    session.StartedAt,
			"components":    session.Registry.List(),
		},
	})
}
