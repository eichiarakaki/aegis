package component

import (
	"sync"
	"time"
)

// Component representa un componente registrado en Aegis
type Component struct {
	ID           string
	SessionID    string
	Name         string
	Version      string
	State        ComponentState
	Capabilities ComponentCapabilities

	StartedAt     time.Time
	LastHeartbeat time.Time
	mu            sync.RWMutex
}

// isValidStateTransition validates if a state transition is allowed
func isValidStateTransition(from, to ComponentState) bool {
	validTransitions := map[ComponentState][]ComponentState{
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
