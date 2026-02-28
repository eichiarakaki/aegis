package sessions

import (
	"fmt"
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// parseRunPayload splits the wire format produced by buildRunPayload:
//
//	<name_or_id>|<mode>|<path1>,<path2>,...
func parseRunPayload(payload string) (nameOrID, mode string, paths []string, err error) {
	parts := strings.SplitN(payload, "|", 3)
	if len(parts) != 3 {
		err = fmt.Errorf("invalid payload: expected <name_or_id>|<mode>|<paths>")
		return
	}

	nameOrID = strings.TrimSpace(parts[0])
	mode = strings.TrimSpace(parts[1])

	for _, p := range strings.Split(parts[2], ",") {
		if p = strings.TrimSpace(p); p != "" {
			paths = append(paths, p)
		}
	}

	if nameOrID == "" {
		err = fmt.Errorf("session name or ID cannot be empty")
	}
	if len(paths) == 0 {
		err = fmt.Errorf("at least one component path is required")
	}
	return
}

// HandleSessionAttach attaches new components to an already running session.
//
// Payload: <name_or_id>||<path1>,<path2>,...
func HandleSessionAttach(payload string, conn net.Conn, sessionStore *core.SessionStore) {
	nameOrID, _, paths, err := parseRunPayload(payload)
	if err != nil {
		writeError(conn, err.Error())
		return
	}

	logger.Infof("Attaching components to session: session=%s paths=%v", nameOrID, paths)

	// TODO:
	//   1. Look up the session by name or ID and retrieve its SessionToken.
	//   2. Verify the session is in a running state.
	//   3. Exec each component binary with AEGIS_SESSION_TOKEN=<token>.

	writeJSON(conn, map[string]interface{}{
		"status":     "ok",
		"session":    nameOrID,
		"components": paths,
	})
}
