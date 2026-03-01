package components

import "github.com/eichiarakaki/aegis/internals/core"

// ComponentDescribe will display all the properties correctly only if it's connected to the aegis-component.sock
func ComponentDescribe(session *core.Session) (map[string]interface{}, error) {
	// TODO: Fix this shii x2

	data := map[string]interface{}{
		"session_id": session.ID,
		"components": session.Components,
	}

	return data, nil
}
