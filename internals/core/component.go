package core

import (
	"sync"
	"time"
)

// Component represents a registered component in Aegis.
type Component struct {
	ID           string                `json:"id"`
	SessionID    string                `json:"session_id"`
	Name         string                `json:"name"`
	Version      string                `json:"version"`
	State        ForeignComponentState `json:"state"`
	Capabilities ComponentCapabilities `json:"capabilities"`

	StartedAt     time.Time    `json:"started_at"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
	mu            sync.RWMutex `json:"-"`
}

func isValidStateTransition(from, to ForeignComponentState) bool {
	validTransitions := map[ForeignComponentState][]ForeignComponentState{
		ComponentStateInit:         {ComponentStateRegistered},
		ComponentStateRegistered:   {ComponentStateInitializing, ComponentStateError},
		ComponentStateInitializing: {ComponentStateReady, ComponentStateError},
		ComponentStateReady:        {ComponentStateConfigured, ComponentStateError},
		ComponentStateConfigured:   {ComponentStateRunning, ComponentStateError},
		ComponentStateRunning:      {ComponentStateWaiting, ComponentStateError, ComponentStateFinished},
		ComponentStateWaiting:      {ComponentStateRunning, ComponentStateError},
		ComponentStateError:        {ComponentStateShutdown},
		ComponentStateFinished:     {ComponentStateShutdown},
		ComponentStateShutdown:     {},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}
	for _, state := range allowed {
		if state == to {
			return true
		}
	}
	return false
}
