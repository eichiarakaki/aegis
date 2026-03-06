package component

import "github.com/eichiarakaki/aegis/internals/core"

// List returns a summary of all components registered in the session.
func List(session *core.Session) core.ComponentListData {
	comps := session.Registry.List()
	list := make([]core.ComponentSummary, 0, len(comps))
	for _, c := range comps {
		list = append(list, core.ComponentSummary{
			ID:                  c.ID,
			Name:                c.Name,
			State:               string(c.State),
			Requires:            requiresMap(c.Capabilities.RequiresStreams),
			SupportedSymbols:    c.Capabilities.SupportedSymbols,
			SupportedTimeframes: c.Capabilities.SupportedTimeframes,
		})
	}
	return core.ComponentListData{SessionID: session.ID, Components: list}
}
