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

	// Control
	StartedAt     time.Time
	LastHeartbeat time.Time
	mu            sync.RWMutex
}

// ComponentRegistry gestiona todos los componentes registrados
type ComponentRegistry struct {
	components map[string]*Component // componentID -> Component
	mu         sync.RWMutex
}

func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		components: make(map[string]*Component),
	}
}

// Register añade un componente al registro
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

// UpdateState actualiza el estado del componente
func (r *ComponentRegistry) UpdateState(componentID string, state ComponentState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	comp, exists := r.components[componentID]
	if !exists {
		return NewValidationError("NOT_FOUND", "component not found")
	}

	// Validar transición de estado
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

// Unregister elimina un componente
func (r *ComponentRegistry) Unregister(componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[componentID]; !exists {
		return NewValidationError("NOT_FOUND", "component not found")
	}

	delete(r.components, componentID)
	return nil
}

// isValidStateTransition valida si una transición de estado es permitida
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
