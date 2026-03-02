package sessions

import (
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
)

func GetSessionState(cmd core.Command, session *core.Session) (core.Response, error) {

	var components []*component.Component

	for _, component := range session.Registry.List() {
		components = append(components, component)
	}

	sessionState := core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_STATE",
		Status:    "ok",
		// ErrorCode: "",
		// Message:   "",
		Data: map[string]interface{}{
			"session":    session,
			"components": components,
		},
	}

	return sessionState, nil
}

// ??? I forgot what I was trying to do here...
//func GetSessionsState(sessionHint string, sessionStore *core.SessionStore, sessionID string) (*core.Session, error) {
//	session, found := GetSessionByHint(sessionID, sessionStore)
//	if !found {
//		return nil, fmt.Errorf("session not found: %s", sessionID)
//	}
//	return session, nil
//}
