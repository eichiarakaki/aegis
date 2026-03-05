package component

import "github.com/eichiarakaki/aegis/internals/core"

// Describe will display all the properties correctly only if it's connected to the aegis-component.sock
func Describe(session *core.Session) (map[string]interface{}, error) {
	// TODO: Fix this shii x2

	data := map[string]interface{}{
		"session_id": session.ID,
		"components": session.Registry.List(),
	}

	return data, nil
}
