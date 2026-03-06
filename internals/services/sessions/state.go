package sessions

import (
	"github.com/eichiarakaki/aegis/internals/core"
)

func GetSessionState(cmd core.Command, session *core.Session) (core.Response, error) {

	var components []*core.Component

	for _, c := range session.Registry.List() {
		components = append(components, c)
	}

	sessionState := core.Response{
		RequestID: cmd.RequestID,
		Command:   core.CommandSessionState,
		Status:    core.OK,
		// ErrorCode: "",
		// Message:   "",
		Data: map[string]any{
			"session":    session,
			"components": components,
		},
	}

	return sessionState, nil
}
