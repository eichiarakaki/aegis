package sessions

import (
	"github.com/eichiarakaki/aegis/internals/core"
)

func GetSessionState(session *core.Session) core.SessionStateData {
	refs := make([]core.ComponentRef, 0)
	for _, c := range session.Registry.List() {
		refs = append(refs, core.ComponentRef{
			Name:  c.Name,
			State: string(c.State),
		})
	}

	return core.SessionStateData{
		Session: core.SessionDetail{
			ID:            session.ID,
			Name:          session.Name,
			Mode:          session.Mode,
			State:         string(session.GetState()),
			UptimeSeconds: session.GetUptimeSeconds(),
			CreatedAt:     session.CreatedAt,
			StartedAt:     session.StartedAt,
			StoppedAt:     session.StoppedAt,
		},
		Components: refs,
	}
}
