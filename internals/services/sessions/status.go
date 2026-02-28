package sessions

import (
	"fmt"

	"github.com/eichiarakaki/aegis/internals/core"
)

func GetStatusByID(sessionHint string, sessionStore *core.SessionStore, sessionID string) (*core.Session, error) {
	session, found := GetSessionByHint(sessionID, sessionStore)
	if !found {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

func GetSessionsStatus(sessionHint string, sessionStore *core.SessionStore, sessionID string) (*core.Session, error) {
	session, found := GetSessionByHint(sessionID, sessionStore)
	if !found {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}
