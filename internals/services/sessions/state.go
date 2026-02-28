package sessions

import (
	"errors"
	"fmt"

	"github.com/eichiarakaki/aegis/internals/core"
)

// StartSession starts a session only if the given session is 'SessionStarting' OR 'SessionStopped'
func StartSession(sessionID string, sessionStore *core.SessionStore) error {
	session, found := GetSessionByHint(sessionID, sessionStore)
	if !found {
		return errors.New("session not found")
	}

	// If the session is starting, already running, or finished: 'start session' is meaningless.
	currentStatus := session.GetStatus()
	// If the session is ALREADY starting, you don't need to start it again.
	if currentStatus == core.SessionStarting {
		return errors.New("session already starting")
	}
	// If the session is ALREADY running, why start it?
	if currentStatus == core.SessionRunning {
		return errors.New("session is already running")
	}
	// If the session is ALREADY finished, you can't start it again without re-making a new session.
	if currentStatus == core.SessionFinished {
		return errors.New("session already started")
	}

	// If the session was just created, you can start it.
	if currentStatus == core.SessionCreated {
		err := session.SetToStarting()
		if err != nil {
			return fmt.Errorf("failed to set session to starting: %w", err)
		}

		// TODO: Here you should call a function to actually start running the session.
	}

	// If the session is currently stopped, you can start it again.
	if currentStatus == core.SessionStopped {
		err := session.SetToRunning()
		if err != nil {
			return fmt.Errorf("failed to set session to running: %w", err)
		}

		// TODO: Here you should call a function to actually start running the session.
	}

	return nil
}
