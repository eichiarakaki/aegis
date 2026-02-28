package sessions

import (
	"fmt"

	"github.com/eichiarakaki/aegis/internals/core"
)

// DeleteSession removes a session by its ID. It returns an error if the session does not exist.
func DeleteSession(sessionID string, sessionStore *core.SessionStore) (string, error) {

	session, found := GetSessionByHint(sessionID, sessionStore)
	if !found {
		return "", fmt.Errorf("there is no unique '%s' session", sessionID)
	}

	id := session.ID

	if session.GetStatus() == core.SessionFinished {
		return id, fmt.Errorf("cannot delete session with ID %s because it is already finished", sessionID)
	}
	if session.GetStatus() == core.SessionRunning {
		return id, fmt.Errorf("cannot delete session with ID %s because it is currently running", sessionID)
	}

	// A session can only be deleted if it is in the stopped state. If it's running, it must be stopped first.

	sessionStore.DeleteSession(session.ID)

	return id, nil
}
