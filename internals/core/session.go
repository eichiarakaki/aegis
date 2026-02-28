package core

import (
	"errors"
	"sync"
	"time"
)

type StatusType int

const (
	SessionCreated StatusType = iota
	SessionStarting
	SessionRunning
	SessionStopping
	SessionStopped
	SessionFinished
)

type Session struct {
	ID     string
	Name   string
	Mode   string // realtime | historical
	Status StatusType

	// Why map instead of slices? O(1) lookups by component name, easier to manage dynamic additions/removals
	Components map[string]*Component

	CreatedAt time.Time
	StartedAt *time.Time
	StoppedAt *time.Time

	mu sync.RWMutex
}

// NewSession creates a new session with the given name and mode. The session starts in the Created state with an empty component list.
func NewSession(id string, name string, mode string) *Session {
	return &Session{
		ID:         id,
		Name:       name,
		Mode:       mode,
		Status:     SessionCreated,
		Components: make(map[string]*Component),
		CreatedAt:  time.Now(),
	}
}

// Start transitions the session to Running state and records the start time. It returns an error if the session is not in a state that can be started.
func (s *Session) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != SessionCreated && s.Status != SessionStopped {
		return errors.New("session cannot be started from current state")
	}

	now := time.Now()
	s.Status = SessionRunning
	s.StartedAt = &now
	return nil
}

// Stop transitions the session to Stopped state and records the stop time. It returns an error if the session is not currently running.
func (s *Session) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != SessionRunning {
		return errors.New("session is not running")
	}

	now := time.Now()
	s.Status = SessionStopped
	s.StoppedAt = &now
	return nil
}

// AddComponent adds a component to the session. Returns an error if the session is finished or if the component already exists.
func (s *Session) AddComponent(c *Component) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == SessionFinished {
		return errors.New("cannot add component to finished session")
	}

	if _, exists := s.Components[*c.Id]; exists {
		return errors.New("component already registered")
	}

	s.Components[*c.Id] = c
	return nil
}

// GetStatus returns the current status of the session. It acquires a read lock to ensure thread safety.
func (s *Session) GetStatus() StatusType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}
