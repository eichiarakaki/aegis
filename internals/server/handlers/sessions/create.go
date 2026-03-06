package sessions

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicessessions "github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleSessionCreate processes SESSION_CREATE commands.
func HandleSessionCreate(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.SessionCreatePayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   err.Error(),
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required fields
	if payload.Name == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session creation failed: empty name")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   "Missing required field: name",
		})
		return
	}

	// Validate mode
	if payload.Mode != "realtime" && payload.Mode != "historical" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session creation failed: invalid mode %s", payload.Mode)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   "Invalid mode: must be 'realtime' or 'historical'",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Creating session: name=%s, mode=%s", payload.Name, payload.Mode)

	// Create the session
	sessionID, err := servicessessions.CreateSession(payload.Name, payload.Mode, sessionStore)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to create session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to create session: %s", err.Error()),
		})
		return
	}

	// Retrieve the created session
	session, found := sessionStore.GetSessionByID(sessionID)
	if !found {
		logger.WithRequestID(cmd.RequestID).Errorf("Session created but not found: %s", sessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   "Session created but could not be retrieved",
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
				"state":      string(session.State),
				"created_at": session.CreatedAt.String(),
			},
		},
	})
}

// HandleSessionCreateRun creates a new session and immediately spawns the provided components.
func HandleSessionCreateRun(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	// Deserialize payload
	var payload core.SessionCreateRunPayload
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to marshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   "Invalid payload format",
		})
		return
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to unmarshal payload: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Payload parsing error: %s", err.Error()),
		})
		return
	}

	// Validate required fields
	if payload.Name == "" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session creation failed: empty name")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   "Missing required field: name",
		})
		return
	}

	if payload.Mode != "realtime" && payload.Mode != "historical" {
		logger.WithRequestID(cmd.RequestID).Warnf("Session creation failed: invalid mode %s", payload.Mode)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   "Invalid mode: must be 'realtime' or 'historical'",
		})
		return
	}

	if len(payload.Paths) == 0 {
		logger.WithRequestID(cmd.RequestID).Warnf("Session creation failed: no component paths provided")
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   "At least one component path is required",
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Creating session and spawning components: name=%s, mode=%s, paths=%d", payload.Name, payload.Mode, len(payload.Paths))

	// Create the session
	sessionID, err := servicessessions.CreateSession(payload.Name, payload.Mode, sessionStore)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to create session: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to create session: %s", err.Error()),
		})
		return
	}

	// Retrieve the created session
	session, found := sessionStore.GetSessionByID(sessionID)
	if !found {
		logger.WithRequestID(cmd.RequestID).Errorf("Session created but not found: %s", sessionID)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   "Session created but could not be retrieved",
		})
		return
	}

	// Attach components
	components, err := servicessessions.AttachComponents(session, payload.Paths)
	if err != nil {
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to attach components: %s", err.Error())
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandSessionCreate,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Failed to attach components: %s", err.Error()),
			Data: map[string]interface{}{
				"session_id": session.ID,
			},
		})
		return
	}

	logger.WithRequestID(cmd.RequestID).Infof("Session created and %d components spawned: %s (%s)", len(components), session.Name, session.ID)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionCreate,
		Status:    core.OK,
		Message:   fmt.Sprintf("Session created and %d components spawned", len(components)),
		Data: map[string]interface{}{
			"session": map[string]interface{}{
				"id":         session.ID,
				"name":       session.Name,
				"mode":       session.Mode,
				"state":      string(session.State),
				"created_at": session.CreatedAt.String(),
			},
			"attached_components": components,
		},
	})
}
