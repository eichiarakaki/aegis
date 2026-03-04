package component

import (
	"sync"
	"time"
)

// ComponentRegistry Manages all the registered components
type ComponentRegistry struct {
	components map[string]*Component // componentID -> Component
	mu         sync.RWMutex
}

func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		components: make(map[string]*Component),
	}
}

// Register adds a component to the register
func (r *ComponentRegistry) Register(comp *Component) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[comp.ID]; exists {
		return NewValidationError("ALREADY_REGISTERED", "component already registered")
	}

	comp.StartedAt = time.Now()
	comp.LastHeartbeat = time.Now()
	r.components[comp.ID] = comp

	return nil
}

// Get obtiene un componente por ID
func (r *ComponentRegistry) Get(componentID string) (*Component, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comp, exists := r.components[componentID]
	return comp, exists
}

// GetBySession obtiene todos los componentes de una sesión
func (r *ComponentRegistry) GetBySession(sessionID string) []*Component {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var components []*Component
	for _, comp := range r.components {
		if comp.SessionID == sessionID {
			components = append(components, comp)
		}
	}

	return components
}

// List returns all registered components.
func (r *ComponentRegistry) List() []*Component {
	r.mu.RLock()
	defer r.mu.RUnlock()

	components := make([]*Component, 0, len(r.components))
	for _, comp := range r.components {
		components = append(components, comp)
	}
	return components
}

// GetByName retrieves a component by name and session.
func (r *ComponentRegistry) GetByName(sessionID, componentName string) (*Component, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, comp := range r.components {
		if comp.SessionID == sessionID && comp.Name == componentName {
			return comp, true
		}
	}

	return nil, false
}

// UpdateState updates the component's state
func (r *ComponentRegistry) UpdateState(componentID string, state ComponentState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	comp, exists := r.components[componentID]
	if !exists {
		return NewValidationError("NOT_FOUND", "component not found")
	}

	// validates the transition
	if !isValidStateTransition(comp.State, state) {
		return NewValidationError("INVALID_STATE_TRANSITION", "invalid state transition")
	}

	comp.State = state
	return nil
}

// GetByState retrieves all components with a specific state.
func (r *ComponentRegistry) GetByState(state ComponentState) []*Component {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var components []*Component
	for _, comp := range r.components {
		if comp.State == state {
			components = append(components, comp)
		}
	}

	return components
}

// UpdateHeartbeat registra el último heartbeat
func (r *ComponentRegistry) UpdateHeartbeat(componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	comp, exists := r.components[componentID]
	if !exists {
		return NewValidationError("NOT_FOUND", "component not found")
	}

	comp.LastHeartbeat = time.Now()
	return nil
}

// Unregister removes a component
func (r *ComponentRegistry) Unregister(componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[componentID]; !exists {
		return NewValidationError("NOT_FOUND", "component not found")
	}

	delete(r.components, componentID)
	return nil
}
