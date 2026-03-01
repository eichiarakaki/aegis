package sessions

import (
	"fmt"
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/services/sessions"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

// parseAttachPayload splits the wire format produced by buildRunPayload:
//
//	<name_or_id>|<mode>|<path1>,<path2>,...
func parseAttachPayload(payload string) (hint, mode string, paths []string, err error) {
	parts := strings.SplitN(payload, "|", 3)
	if len(parts) != 3 {
		err = fmt.Errorf("invalid payload: expected <name_or_id>|<mode>|<paths>")
		return
	}

	hint = strings.TrimSpace(parts[0])
	mode = strings.TrimSpace(parts[1])

	for _, p := range strings.Split(parts[2], ",") {
		if p = strings.TrimSpace(p); p != "" {
			paths = append(paths, p)
		}
	}

	if hint == "" {
		err = fmt.Errorf("invalid payload: missing <name_or_id>|<mode>")
	}
	if len(paths) == 0 {
		err = fmt.Errorf("at least one component path is required")
	}
	return
}

// HandleSessionAttach attaches new components to an already running session.
//
// Payload: <name_or_id>||<path1>,<path2>,...
func HandleSessionAttach(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {
	hint, _, paths, err := parseAttachPayload(cmd.Payload)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			// ErrorCode: "",
			Message: fmt.Sprint("Attach failed:", err.Error()),
			Data: map[string]string{
				"session_id": hint,
			},
		})
		return
	}

	session, found := sessions.GetSessionByHint(hint, sessionStore)
	if !found {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			// ErrorCode: "",
			Message: fmt.Sprint("Couldn't find the session"),
			Data: map[string]string{
				"session_id": hint,
			},
		})
		return
	}

	components, err := sessions.AttachComponents(session, paths)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   "SESSION_ATTACH",
			Status:    "error",
			// ErrorCode: "",
			Message: fmt.Sprint("Attach failed:", err.Error()),
			Data: map[string]string{
				"session_id": hint,
			},
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_ATTACH",
		Status:    "ok",
		//ErrorCode: "",
		Message: fmt.Sprintf("Attached %v components to %s (%s)", components, session.Name, utils.GetShortHash(session.ID)),
		Data: map[string]interface{}{
			"session_id":          session.ID,
			"attached_components": components,
		},
	})
}
