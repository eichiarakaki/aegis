package core

import (
	"sync"
	"time"
)

// Registry Manages all the registered components
type Registry struct {
	components map[string]*Component // componentID -> Component
	mu         sync.RWMutex
}

func NewComponentRegistry() *Registry {
	return &Registry{
		components: make(map[string]*Component),
	}
}

// Register adds a component to the register
func (r *Registry) Register(comp *Component) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[comp.ID]; exists {
		return NewValidationError(string(CommandAlreadyRegistered), "component already registered")
	}

	comp.StartedAt = time.Now()
	comp.LastHeartbeat = time.Now()
	r.components[comp.ID] = comp

	return nil
}

// Get gets a component by ID
func (r *Registry) Get(componentID string) (*Component, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comp, exists := r.components[componentID]
	return comp, exists
}

// GetBySession gets all the components of a session
func (r *Registry) GetBySession(sessionID string) []*Component {
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
func (r *Registry) List() []*Component {
	r.mu.RLock()
	defer r.mu.RUnlock()

	components := make([]*Component, 0, len(r.components))
	for _, comp := range r.components {
		components = append(components, comp)
	}
	return components
}

// Count the amount of registered.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.components)
}

// GetByName retrieves a component by name and session.
func (r *Registry) GetByName(sessionID, componentName string) (*Component, bool) {
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
func (r *Registry) UpdateState(componentID string, state ForeignComponentState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	comp, exists := r.components[componentID]
	if !exists {
		return NewValidationError(string(NOT_FOUND), "component not found")
	}

	// validates the transition
	if !isValidStateTransition(comp.State, state) {
		return NewValidationError(string(INVALID_STATE_TRANSITION), "invalid state transition")
	}

	comp.State = state
	return nil
}

// GetByState retrieves all components with a specific state.
func (r *Registry) GetByState(state ForeignComponentState) []*Component {
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

// UpdateHeartbeat registers the last heartbeat
func (r *Registry) UpdateHeartbeat(componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	comp, exists := r.components[componentID]
	if !exists {
		return NewValidationError(string(NOT_FOUND), "component not found")
	}

	comp.LastHeartbeat = time.Now()
	return nil
}

// Unregister removes a component
func (r *Registry) Unregister(componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[componentID]; !exists {
		return NewValidationError(string(NOT_FOUND), "component not found")
	}

	delete(r.components, componentID)
	return nil
}
