package component

import "github.com/eichiarakaki/aegis/internals/core"

// List will display all the properties correctly only if it's connected to the aegis-component.sock
func List(session *core.Session) (map[string]any, error) {

	data := map[string]any{
		"session_id": session.ID,
		"components": session.Registry.List(),
	}

	return data, nil
}
