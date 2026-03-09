package sessions

import (
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

	logger.WithRequestID(cmd.RequestID).Infof("Starting session: %s (from=%d to=%d)", payload.SessionID, payload.From, payload.To)

	session, err := sessions.GetSessionByHint(payload.SessionID, sessionStore)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStart,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	previousState := session.State

	tr := sessions.TimeRange{From: payload.From, To: payload.To}
	if err := sessions.StartSession(session, cmd, conn, nc, tr); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to start session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionStart,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to start session: %s", err.Error()),
			Data: map[string]any{
				"session_id":     session.ID,
				"previous_state": string(previousState),
				"current_state":  string(core.SessionError),
			},
		})
		return
	}

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
