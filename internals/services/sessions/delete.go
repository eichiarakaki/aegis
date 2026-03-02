package sessions

import (
	"fmt"

	"github.com/eichiarakaki/aegis/internals/core"
)

// DeleteSession removes a session only is the session is stopped, finished or initialized.
func DeleteSession(session *core.Session, sessionStore *core.SessionStore) error {
	// Filtering everything else but these states
	if session.GetState() != core.SessionStopped && session.GetState() != core.SessionFinished && session.GetState() != core.SessionInitialized {
		return fmt.Errorf("can't delete session from the current state (%s). Session must be Stopped, Finished or Initialized", core.SessionStateToString(session.GetState()))
	}

	sessionStore.DeleteSession(session.ID)

	return nil
}
