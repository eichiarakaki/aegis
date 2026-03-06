package sessions

import (
	"github.com/eichiarakaki/aegis/internals/core"
)

func ListSessions(sessionStore *core.SessionStore) core.SessionListData {
	sessions := sessionStore.ListSessions()

	entries := make([]core.SessionListEntry, 0, len(sessions))
	for _, s := range sessions {
		count := 0
		if s.Registry != nil {
			count = len(s.Registry.List())
		}
		entries = append(entries, core.SessionListEntry{
			ID:             s.ID,
			Name:           s.Name,
			Mode:           s.Mode,
			State:          string(s.GetState()),
			CreatedAt:      s.CreatedAt,
			StartedAt:      s.StartedAt,
			StoppedAt:      s.StoppedAt,
			ComponentCount: count,
		})
	}

	return core.SessionListData{Sessions: entries}
}
