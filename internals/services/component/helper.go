package component

import (
	"fmt"
	"strings"

	"github.com/eichiarakaki/aegis/internals/core"
)

// ── resolveComponent ──────────────────────────────────────────────────────────

// resolveComponent resolves ref to a component using this priority:
//  1. Empty ref + exactly one component → use it
//  2. Exact ID match
//  3. Exact name match
//  4. ID prefix match (unique)
//  5. Name prefix match (unique)
//  6. Name substring match (unique)
func resolveComponent(session *core.Session, ref string) (*core.Component, error) {
	all := session.Registry.List()

	if ref == "" {
		switch len(all) {
		case 0:
			return nil, fmt.Errorf("session %s has no components", session.ID)
		case 1:
			return all[0], nil
		default:
			return nil, fmt.Errorf("session %s has %d components, specify one: %s",
				session.ID, len(all), componentNames(all))
		}
	}

	lower := strings.ToLower(ref)

	for _, c := range all {
		if c.ID == ref {
			return c, nil
		}
	}
	for _, c := range all {
		if strings.ToLower(c.Name) == lower {
			return c, nil
		}
	}

	var idMatches, prefixMatches, subMatches []*core.Component
	for _, c := range all {
		if strings.HasPrefix(strings.ToLower(c.ID), lower) {
			idMatches = append(idMatches, c)
		}
	}
	if len(idMatches) == 1 {
		return idMatches[0], nil
	}
	if len(idMatches) > 1 {
		return nil, fmt.Errorf("ambiguous id prefix %q: %s", ref, componentNames(idMatches))
	}

	for _, c := range all {
		if strings.HasPrefix(strings.ToLower(c.Name), lower) {
			prefixMatches = append(prefixMatches, c)
		}
	}
	if len(prefixMatches) == 1 {
		return prefixMatches[0], nil
	}
	if len(prefixMatches) > 1 {
		return nil, fmt.Errorf("ambiguous name prefix %q: %s", ref, componentNames(prefixMatches))
	}

	for _, c := range all {
		if strings.Contains(strings.ToLower(c.Name), lower) {
			subMatches = append(subMatches, c)
		}
	}
	if len(subMatches) == 1 {
		return subMatches[0], nil
	}
	if len(subMatches) > 1 {
		return nil, fmt.Errorf("ambiguous name %q: %s", ref, componentNames(subMatches))
	}

	return nil, fmt.Errorf("component %q not found in session %s", ref, session.ID)
}

func componentNames(comps []*core.Component) string {
	names := make([]string, len(comps))
	for i, c := range comps {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}

func requiresMap(streams []string) map[string]bool {
	m := make(map[string]bool, len(streams))
	for _, s := range streams {
		m[s] = true
	}
	return m
}
