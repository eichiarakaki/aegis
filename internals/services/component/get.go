package component

import "github.com/eichiarakaki/aegis/internals/core"

// Get will display all the properties correctly only if it's connected to the aegis-component.sock
func Get(session *core.Session, componentID string) (map[string]any, error) {
	// TODO: Fix this shii

	data := map[string]any{
		"session_id": session.ID,
		"components": session.Registry.List(),
	}

	return data, nil
}
