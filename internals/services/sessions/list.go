package sessions

import "github.com/eichiarakaki/aegis/internals/core"

func ListSessions(sessionStore *core.SessionStore) map[string]any {
	sessions := sessionStore.ListSessions()

	result := make(map[string]any)
	for _, session := range sessions {
		result[session.ID] = map[string]any{
			"id":         session.ID,
			"name":       session.Name,
			"mode":       session.Mode,
			"state":      core.SessionStateToString(session.GetStatus()),
			"components": session.Components,
			"created_at": session.CreatedAt,
			"started_at": session.StartedAt,
			"stopped_at": session.StoppedAt,
		}
	}

	return result
}
