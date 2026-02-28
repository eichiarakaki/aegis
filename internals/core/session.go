package core

import (
	"errors"
	"fmt"
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

// SetToRunning sets to start...
func (s *Session) SetToRunning() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != SessionStarting && s.Status != SessionStopped {
		return errors.New("session cannot be started from the current state")
	}

	now := time.Now()
	s.Status = SessionRunning
	s.StartedAt = &now
	return nil
}

// SetToStarting sets to start...
func (s *Session) SetToStarting() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != SessionCreated {
		return errors.New("session cannot be started from the current state")
	}

	now := time.Now()
	s.Status = SessionStarting
	s.StartedAt = &now
	return nil
}

// SetToStop Stop transitions the session to Stopped state and records the stop time. It returns an error if the session is not currently running.
func (s *Session) SetToStop() error {
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

	if _, exists := s.Components[*c.ID]; exists {
		return errors.New("component already registered")
	}

	s.Components[*c.ID] = c
	return nil
}

// GetStatus returns the current status of the session. It acquires a read lock to ensure thread safety.
func (s *Session) GetStatus() StatusType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// SessionStore manages all active sessions in memory. It provides thread-safe methods to add, retrieve, list, and delete sessions.
type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

func (store *SessionStore) AddSession(s *Session) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, exists := store.sessions[s.ID]
	if exists {
		return fmt.Errorf("session with the same ID already exists")
	}
	store.sessions[s.ID] = s
	return nil
}

func (store *SessionStore) GetSessionByID(id string) (*Session, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	s, exists := store.sessions[id]
	return s, exists
}

func (store *SessionStore) GetSessionByName(name string) (*Session, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, s := range store.sessions {
		if s.Name == name {
			return s, true
		}
	}
	return nil, false
}

func (store *SessionStore) GetSessionsByName(name string) ([]*Session, int) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	var sessions []*Session
	count := 0
	for _, s := range store.sessions {
		if s.Name == name {
			sessions = append(sessions, s)
			count++
		}
	}
	if len(sessions) >= 1 {
		return sessions, count
	}

	return nil, 0
}

func (store *SessionStore) GetSessionsByMode(mode string) []*Session {
	store.mu.RLock()
	defer store.mu.RUnlock()
	var sessions []*Session
	for _, s := range store.sessions {
		if s.Mode == mode {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

func (store *SessionStore) GetSessionsByStatus(status StatusType) []*Session {
	store.mu.RLock()
	defer store.mu.RUnlock()
	var sessions []*Session
	for _, s := range store.sessions {
		if s.GetStatus() == status {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// GetSessionByIDApproximation allows retrieval of a session by its ID or by an approximation of the ID (first 4 characters). This is useful for user-friendly commands where the full ID may be cumbersome to type. It returns the session and a boolean indicating if it was found.
func (store *SessionStore) GetSessionByIDApproximation(id string) (*Session, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, s := range store.sessions {
		if s.ID == id || (len(id) >= 4 && len(s.ID) >= 4 && s.ID[:4] == id[:4]) {
			return s, true
		}
	}
	return nil, false
}

func (store *SessionStore) ListSessions() []*Session {
	store.mu.RLock()
	defer store.mu.RUnlock()
	sessions := make([]*Session, 0, len(store.sessions))
	for _, s := range store.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

func (store *SessionStore) DeleteSession(id string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.sessions, id)
}
