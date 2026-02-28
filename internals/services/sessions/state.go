package sessions

import (
	"fmt"

	"github.com/eichiarakaki/aegis/internals/core"
)

func GetSessionStateByID(sessionHint string, sessionStore *core.SessionStore) (string, error) {
	session, found := GetSessionByHint(sessionHint, sessionStore)
	if !found {
		return "", fmt.Errorf("session not found: %s", sessionHint)
	}
	return core.SessionStateToString(session.State), nil
}

// ??? I forgot what I was trying to do here...
//func GetSessionsState(sessionHint string, sessionStore *core.SessionStore, sessionID string) (*core.Session, error) {
//	session, found := GetSessionByHint(sessionID, sessionStore)
//	if !found {
//		return nil, fmt.Errorf("session not found: %s", sessionID)
//	}
//	return session, nil
//}
