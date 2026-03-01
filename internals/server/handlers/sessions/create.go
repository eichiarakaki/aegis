package sessions

import (
	"fmt"
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	services_sessions "github.com/eichiarakaki/aegis/internals/services/sessions"
)

// HandleSessionCreate processes SESSION_CREATE commands.
// Payload format: "<name>|<mode>"
func HandleSessionCreate(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	parts := strings.SplitN(cmd.Payload, "|", 2)
	if len(parts) != 2 {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_CREATE",
			Status:    "error",
			Message:   fmt.Sprintf("invalid payload: %s", cmd.Payload),
			Data:      map[string]string{}})
		return
	}

	name, mode := parts[0], parts[1]

	if strings.TrimSpace(name) == "" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_CREATE",
			Status:    "error",
			Message:   fmt.Sprintf("Empty payload"),
			Data:      map[string]string{}})
		logger.WithRequestID(cmd.RequestID).Debugf("Empty payload: %s", cmd.Payload)
		return
	}

	if mode != "realtime" && mode != "historical" {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_CREATE",
			Status:    "error",
			Message:   fmt.Sprintf("Unsupported Mode"),
			Data:      map[string]string{}})
		logger.WithRequestID(cmd.RequestID).Debugf("Unsupported Mode: %s", cmd.Payload)
		return
	}

	// Creating the session
	id, err := services_sessions.CreateSession(name, mode, sessionStore)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_CREATE",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to create session: %s", err.Error()),
			Data:      map[string]string{}})
		logger.WithRequestID(cmd.RequestID).Errorf("Failed to create session: %s", err.Error())
		return
	}

	// Getting the created session
	session, found := sessionStore.GetSessionByID(id)
	if !found {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_CREATE",
			Status:    "error",
			Message:   fmt.Sprintf("Failed to get the created session: Session not found"),
			Data:      map[string]string{}})
		logger.WithRequestID(cmd.RequestID).Error("Failed to get the created session: Session not found")
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_CREATE",
		Status:    "ok",
		Message:   fmt.Sprintf("Session was created successfully: %s", session.ID),
		Data: map[string]map[string]string{
			"session": {
				"id":         session.ID,
				"name":       session.Name,
				"mode":       session.Mode,
				"state":      core.SessionStateToString(session.State),
				"created_at": session.CreatedAt.String(),
			},
		}})
}

// HandleSessionCreateRun creates a new session and immediately spawns
// the provided components under a fresh SessionToken.
//
// Payload: <n>|<mode>|<path1>,<path2>,...
//func HandleSessionCreateRun(payload string, conn net.Conn, sessionStore *core.SessionStore) {
//	name, mode, paths, err := parseRunPayload(payload)
//	if err != nil {
//		writeError(conn, err.Error())
//		return
//	}
//
//	if mode != "realtime" && mode != "historical" {
//		writeError(conn, "invalid mode: must be 'realtime' or 'historical'")
//		return
//	}
//
//	logger.Infof("Creating session and running components: name=%s mode=%s paths=%v", name, mode, paths)
//
//	// TODO:
//	//   1. Persist the new session record and generate a SessionToken.
//	//   2. For each path, exec the binary with AEGIS_SESSION_TOKEN=<token>.
//	//   3. Components connect to /tmp/aegis-data-stream-<session_id>.sock
//	//      once the token is verified.
//
//	writeJSON(conn, map[string]interface{}{
//		"status":     "ok",
//		"session":    name,
//		"mode":       mode,
//		"components": paths,
//	})
//}
