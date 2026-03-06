package component

import (
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
)

// Get returns detailed info for a single component.
// ref can be an exact ID, exact name, ID prefix, or name prefix/substring.
// If ref is empty and the session has exactly one component, it is used automatically.
func Get(session *core.Session, ref string) (core.ComponentGetData, error) {
	c, err := resolveComponent(session, ref)
	if err != nil {
		return core.ComponentGetData{SessionID: session.ID}, err
	}

	var uptime int64
	if !c.StartedAt.IsZero() {
		uptime = int64(time.Since(c.StartedAt).Seconds())
	}

	return core.ComponentGetData{
		SessionID: session.ID,
		Component: core.ComponentDetail{
			ID:                  c.ID,
			Name:                c.Name,
			State:               string(c.State),
			Requires:            requiresMap(c.Capabilities.RequiresStreams),
			SupportedSymbols:    c.Capabilities.SupportedSymbols,
			SupportedTimeframes: c.Capabilities.SupportedTimeframes,
			StartedAt:           c.StartedAt,
			UptimeSeconds:       uptime,
		},
	}, nil
}
