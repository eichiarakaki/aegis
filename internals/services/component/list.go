package component

import "github.com/eichiarakaki/aegis/internals/core"

// ComponentList will display all the properties correctly only if it's connected to the aegis-component.sock
func ComponentList(session *core.Session) (map[string]interface{}, error) {

	data := map[string]interface{}{
		"session_id": session.ID,
		"components": session.Components,
	}

	return data, nil
}
