package sessions

import (
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicessessions "github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleSessionCreate processes SESSION_CREATE commands.
func HandleSessionCreate(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	payload, err := core.DeserializeSessionCreatePayload(cmd)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to execute your command: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Creating session: name=%s, mode=%s, market=%s",
		payload.Name, payload.Mode, payload.Market)

	session, err := sessionCreation(payload.Name, payload.Mode, payload.Market, sessionStore)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to create session: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session created successfully: %s (%s)", session.Name, session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionCreate,
		Status:    core.OK,
		Message:   fmt.Sprintf("Session created successfully: %s", session.Name),
		Data: map[string]interface{}{
			"session": map[string]interface{}{
				"id":         session.ID,
				"name":       session.Name,
				"mode":       session.Mode,
				"market":     session.Market,
				"state":      session.State,
				"created_at": session.CreatedAt.String(),
			},
		},
	})
}

// HandleSessionCreateRun creates a new session and immediately spawns the provided components.
func HandleSessionCreateRun(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	payload, err := core.DeserializeSessionCreateRunPayload(cmd)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreateRun,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to deserialize your command: %s", err.Error()),
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Creating session and spawning components: name=%s, mode=%s, market=%s, paths=%d",
		payload.Name, payload.Mode, payload.Market, len(payload.Paths))

	session, err := sessionCreation(payload.Name, payload.Mode, payload.Market, sessionStore)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreateRun,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to create session: %s", err.Error()),
		})
		return
	}

	components, err := servicessessions.AttachComponents(session, payload.Paths)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to attach components: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreateRun,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to attach components: %s", err.Error()),
			Data: map[string]interface{}{
				"session_id": session.ID,
			},
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session created and %d components spawned: %s (%s)",
		len(components), session.Name, session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionCreateRun,
		Status:    core.OK,
		Message:   fmt.Sprintf("Session created and %d components spawned", len(components)),
		Data: map[string]interface{}{
			"session": map[string]any{
				"id":         session.ID,
				"name":       session.Name,
				"mode":       session.Mode,
				"market":     session.Market,
				"state":      string(session.State),
				"created_at": session.CreatedAt.String(),
			},
			"attached_components": components,
		},
	})
}

func sessionCreation(name, mode, market string, sessionStore *core.SessionStore) (*core.Session, error) {
	sessionID, err := servicessessions.CreateSession(name, mode, market, sessionStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %s", err.Error())
	}

	session, found := sessionStore.GetSessionByID(sessionID)
	if !found {
		return nil, fmt.Errorf("session created but could not be retrieved: %s", sessionID)
	}

	return session, nil
}
