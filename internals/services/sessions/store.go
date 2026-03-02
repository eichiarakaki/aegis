package sessions

import (
	"fmt"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// GetSessionByHint resolves a session by name, ID approximation, or full ID.
// It returns the session and a boolean indicating if it was found.
// Priority: full ID > ID approximation > name
func GetSessionByHint(hint string, sessionStore *core.SessionStore) (*core.Session, error) {
	if hint == "" {
		logger.Warn("GetSessionByHint: empty hint provided")
		return nil, fmt.Errorf("empty hint provided")
	}

	logger.WithComponent("sessions").Debugf("Resolving session hint: %s", hint)

	// Try full ID first (highest priority, most specific)
	if session, found := sessionStore.GetSessionByID(hint); found {
		logger.WithComponent("sessions").Debugf("Session resolved by full ID: %s", session.ID)
		return session, nil
	}

	// Try ID approximation (first N characters)
	if session, found := sessionStore.GetSessionByIDApproximation(hint); found {
		logger.WithComponent("sessions").Debugf("Session resolved by ID approximation: %s", session.ID)
		return session, nil
	}

	// Try name (lowest priority, may have collisions)
	sessions, count := sessionStore.GetSessionsByName(hint)
	if count > 1 {
		logger.WithComponent("sessions").Warnf("Multiple sessions found with name '%s' (%d matches)", hint, count)
		return nil, fmt.Errorf("multiple sessions found with name '%s' (%d matches). Use session ID for disambiguation", hint, count)
	}

	if count == 1 && sessions != nil && len(sessions) > 0 {
		logger.WithComponent("sessions").Debugf("Session resolved by name: %s", sessions[0].Name)
		return sessions[0], nil
	}

	logger.WithComponent("sessions").Warnf("Session not found: %s", hint)
	return nil, fmt.Errorf("session not found: %s", hint)
}
