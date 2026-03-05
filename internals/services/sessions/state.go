package sessions

import (
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
)

func GetSessionState(cmd core.Command, session *core.Session) (core.Response, error) {

	var components []*component.Component

	for _, c := range session.Registry.List() {
		components = append(components, c)
	}

	sessionState := core.Response{
		RequestID: cmd.RequestID,
		Command:   "SESSION_STATE",
		Status:    "ok",
		// ErrorCode: "",
		// Message:   "",
		Data: map[string]any{
			"session":    session,
			"components": components,
		},
	}

	return sessionState, nil
}
