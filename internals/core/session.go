package core

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type SessionStateType int

const (
	SessionInitialized SessionStateType = iota
	SessionStarting
	SessionRunning
	SessionStopping
	SessionStopped
	SessionFinished
	SessionError
)

func SessionStateToString(state SessionStateType) string {
	switch state {
	case SessionInitialized:
		return "initialized"
	case SessionStarting:
		return "starting"
	case SessionRunning:
		return "running"
	case SessionStopping:
		return "stopping"
	case SessionStopped:
		return "stopped"
	case SessionFinished:
		return "finished"
	case SessionError:
		return "error"
	default:
		return "unknown"
	}
}

type Session struct {
	ID         string
	Name       string
	Mode       string // realtime | historical
	State      SessionStateType
	Components map[string]*Component

	CreatedAt time.Time
	StartedAt *time.Time
	StoppedAt *time.Time

	mu sync.RWMutex
}

// NewSession creates a new session with the given name and mode.
func NewSession(id string, name string, mode string) *Session {
	return &Session{
		ID:         id,
		Name:       name,
		Mode:       mode,
		State:      SessionInitialized,
		Components: make(map[string]*Component),
		CreatedAt:  time.Now(),
	}
}

// GetUptimeSeconds calculates and returns the uptime in seconds.
// Returns 0 if the session has not started yet.
// If the session is running or was stopped, returns the duration between StartedAt and StoppedAt (or now).
func (s *Session) GetUptimeSeconds() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Session hasn't started yet
	if s.StartedAt == nil {
		return 0
	}

	// Session is running or in an intermediate state
	if s.StoppedAt == nil {
		return int64(time.Since(*s.StartedAt).Seconds())
	}

	// Session has stopped, calculate duration between start and stop
	return int64(s.StoppedAt.Sub(*s.StartedAt).Seconds())
}

// SetToRunning transitions the session from SessionStarting to SessionRunning.
func (s *Session) SetToRunning() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != SessionStarting && s.State != SessionStopped {
		return errors.New("session cannot transition to running from current state")
	}

	now := time.Now()
	s.State = SessionRunning
	s.StartedAt = &now
	return nil
}

// SetToStarting transitions the session from SessionInitialized to SessionStarting.
func (s *Session) SetToStarting() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != SessionInitialized {
		return errors.New("session cannot transition to starting from current state")
	}

	now := time.Now()
	s.State = SessionStarting
	s.StartedAt = &now
	return nil
}

// SetToStop transitions the session from SessionRunning to SessionStopped.
// It records the stop time and calculates the uptime.
func (s *Session) SetToStop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != SessionRunning && s.State != SessionStarting {
		return errors.New("session is not running or starting")
	}

	now := time.Now()
	s.State = SessionStopped
	s.StoppedAt = &now
	return nil
}

// AddComponent adds a component to the session.
// Returns an error if the session is finished or if the component already exists.
func (s *Session) AddComponent(c *Component) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == SessionFinished {
		return errors.New("cannot add component to finished session")
	}

	if _, exists := s.Components[*c.ID]; exists {
		return errors.New("component already registered")
	}

	s.Components[*c.ID] = c
	return nil
}

// GetState returns the current state of the session.
func (s *Session) GetState() SessionStateType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// SessionStore manages all active sessions in memory.
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

func (store *SessionStore) GetSessionsByStatus(state SessionStateType) []*Session {
	store.mu.RLock()
	defer store.mu.RUnlock()
	var sessions []*Session
	for _, s := range store.sessions {
		if s.GetState() == state {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// GetSessionByIDApproximation retrieves a session by its full ID or by an approximation.
func (store *SessionStore) GetSessionByIDApproximation(approximation string) (*Session, bool) {
	if approximation == "" {
		return nil, false
	}

	const minApproximationLength = 4

	store.mu.RLock()
	defer store.mu.RUnlock()

	// First, try exact match
	for _, session := range store.sessions {
		if session.ID == approximation {
			return session, true
		}
	}

	// Then, try approximation if length is sufficient
	if len(approximation) >= minApproximationLength {
		for _, session := range store.sessions {
			if len(session.ID) >= minApproximationLength && session.ID[:minApproximationLength] == approximation[:minApproximationLength] {
				return session, true
			}
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

func (store *SessionStore) Count() int {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return len(store.sessions)
}

func (store *SessionStore) CountByState(state SessionStateType) int {
	store.mu.RLock()
	defer store.mu.RUnlock()

	count := 0
	for _, session := range store.sessions {
		if session.State == state {
			count++
		}
	}
	return count
}

func (store *SessionStore) TotalComponents() int {
	store.mu.RLock()
	defer store.mu.RUnlock()

	count := 0
	for _, session := range store.sessions {
		count += len(session.Components)
	}

	return count
}

func (store *SessionStore) TotalComponentsByStateFromAllSessions(state ComponentStateType) int {
	store.mu.RLock()
	defer store.mu.RUnlock()
	count := 0

	for _, session := range store.sessions {
		for _, component := range session.Components {
			if component.State == state {
				count++
			}
		}
	}

	return count
}
