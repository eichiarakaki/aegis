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

// HandleSessionResume resumes a STOPPED session.
// Resume differs from restart: it is only valid from STOPPED (not FINISHED),
// and does not accept a new time range — it continues from where it left off.
// The state machine already allows STOPPED → STARTING.
func HandleSessionResume(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore, nc *nats.Conn, logStore *servicescomponent.LogStore) {
	payload, err := core.DeserializeSessionActionPayload(cmd)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionResume,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	session, err := sessionStore.GetByHint(payload.SessionID)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionResume,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	if session.GetState() != core.SessionStopped {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionResume,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("resume is only valid for STOPPED sessions (current state: %s)", session.GetState()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Resuming session: %s", payload.SessionID)

	previousState := session.State

	// Resume passes zero TimeRange — no range filtering, continues with same topics.
	tr := sessions.TimeRange{}
	if err := sessions.StartSession(session, cmd, conn, nc, tr); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to resume session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionResume,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to resume session: %s", err.Error()),
			Data: map[string]any{
				"session_id":     session.ID,
				"previous_state": previousState,
			},
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionResume,
		Status:    core.OK,
		Message:   fmt.Sprintf("Session resumed: %s", utils.GetShortHash(session.ID)),
		Data: map[string]any{
			"session_id":     session.ID,
			"previous_state": previousState,
			"current_state":  session.State,
			"started_at":     session.StartedAt,
			"components":     session.Registry.List(),
		},
	})
}
