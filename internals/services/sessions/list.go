package sessions

import (
	"github.com/eichiarakaki/aegis/internals/core"
)

func ListSessions(sessionStore *core.SessionStore) map[string]any {
	sessions := sessionStore.ListSessions()

	result := make(map[string]any)
	for _, session := range sessions {

		// Read components from the session's registry
		var componentList []map[string]any
		if session.Registry != nil {
			for _, comp := range session.Registry.List() {
				componentList = append(componentList, map[string]any{
					"id":             comp.ID,
					"name":           comp.Name,
					"version":        comp.Version,
					"state":          comp.State,
					"started_at":     comp.StartedAt,
					"last_heartbeat": comp.LastHeartbeat,
				})
			}
		}

		if componentList == nil {
			componentList = []map[string]any{} // avoid null in JSON
		}

		result[session.ID] = map[string]any{
			"id":            session.ID,
			"name":          session.Name,
			"stream_socket": session.GetStreamSocketPath(),
			"topics":        session.Topics,
			"mode":          session.Mode,
			"state":         core.SessionStateToString(session.GetState()),
			"components":    componentList,
			"created_at":    session.CreatedAt,
			"started_at":    session.StartedAt,
			"stopped_at":    session.StoppedAt,
		}
	}
	return result
}
