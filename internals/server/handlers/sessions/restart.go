package sessions

import (
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/utils"
	"github.com/nats-io/nats.go"
)

// HandleSessionRestart restarts a FINISHED session without relaunching component processes.
func HandleSessionRestart(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore, nc *nats.Conn) {
	payload, err := core.DeserializeSessionStartPayload(cmd)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionRestart,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	session, err := sessionStore.GetByHint(payload.SessionID)
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
