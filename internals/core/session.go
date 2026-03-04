package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/core/component"
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
		return "INITIALIZED"
	case SessionStarting:
		return "STARTING"
	case SessionRunning:
		return "RUNNING"
	case SessionStopping:
		return "STOPPING"
	case SessionStopped:
		return "STOOPED"
	case SessionFinished:
		return "FINISHED"
	case SessionError:
		return "ERROR"
	default:
		return "UNKNOW"
	}
}

type Session struct {
	ID       string
	Name     string
	Mode     string // realtime | historical
	State    SessionStateType
	Registry *component.ComponentRegistry

	CreatedAt time.Time
	StartedAt *time.Time
	StoppedAt *time.Time

	mu sync.RWMutex
}

// NewSession creates a new session with the given name and mode.
func NewSession(id string, name string, mode string) *Session {
	return &Session{
		ID:        id,
		Name:      name,
		Mode:      mode,
		State:     SessionInitialized,
		Registry:  component.NewComponentRegistry(),
		CreatedAt: time.Now(),
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

	if !IsValidSessionStateTransition(s.State, SessionRunning) {
		return errors.New("session cannot transition to running from the current state")
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

	if !IsValidSessionStateTransition(s.State, SessionStarting) {
		return errors.New("session cannot transition to starting from current state")
	}

	now := time.Now()
	s.State = SessionStarting
	s.StartedAt = &now
	return nil
}

// SetToStopping transitions the session from SessionRunning to SessionStopping.
func (s *Session) SetToStopping() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !IsValidSessionStateTransition(s.State, SessionStopping) {
		return errors.New("session cannot transition to stopping from current state")
	}

	now := time.Now()
	s.State = SessionStopping
	s.StartedAt = &now
	return nil
}

// SetToStopped transitions the session from SessionStopping to SessionStopped.
func (s *Session) SetToStopped() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !IsValidSessionStateTransition(s.State, SessionStopped) {
		return errors.New("session cannot transition to stopped from current state")
	}

	now := time.Now()
	s.State = SessionStopped
	s.StartedAt = &now
	return nil
}

// AddComponent adds a component to the session.
// Returns an error if the session is finished or if the component already exists.
func (s *Session) AddComponent(c *component.Component) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == SessionFinished {
		return errors.New("cannot add component to finished session")
	}

	if s.Registry == nil {
		return errors.New("session registry is not initialized")
	}

	return s.Registry.Register(c)
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

func (store *SessionStore) IsTokenInSessionStore(token string) bool {
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, session := range store.sessions {
		if session.ID == token {
			return true
		}
	}
	return false
}

// GetSessionByIDApproximation retrieves a session by its full ID or by a dynamic approximation.
// The approximation uses progressively longer prefixes of the session ID.
// Examples:
//   - "abc" matches "abcdef123..." if it's unique
//   - If multiple sessions start with "abc", it returns nil (ambiguous)
//   - Full ID always takes priority
//
// Returns the session and a boolean indicating if it was found.
func (store *SessionStore) GetSessionByIDApproximation(approximation string) (*Session, bool) {
	if approximation == "" {
		return nil, false
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	// First, try exact match (full ID)
	for _, session := range store.sessions {
		if session.ID == approximation {
			return session, true
		}
	}

	// Then, try dynamic prefix matching
	// Start from approximation length and go up to the max ID length
	maxIDLength := 0
	for _, session := range store.sessions {
		if len(session.ID) > maxIDLength {
			maxIDLength = len(session.ID)
		}
	}

	// Try progressively longer prefixes
	for prefixLen := len(approximation); prefixLen <= maxIDLength; prefixLen++ {
		if prefixLen > len(approximation) {
			break // Don't search beyond the approximation length on first pass
		}

		matches := 0
		var matchedSession *Session

		for _, session := range store.sessions {
			if len(session.ID) >= prefixLen && session.ID[:prefixLen] == approximation[:prefixLen] {
				matches++
				matchedSession = session
				if matches > 1 {
					break // Ambiguous: multiple matches
				}
			}
		}

		// If exactly one match at this prefix length, return it
		if matches == 1 {
			return matchedSession, true
		}

		// If multiple matches, stop searching (ambiguous)
		if matches > 1 {
			return nil, false
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
		if session.Registry == nil {
			continue
		}
		count += len(session.Registry.List())
	}

	return count
}

func (store *SessionStore) TotalComponentsByStateFromAllSessions(state component.ComponentState) int {
	store.mu.RLock()
	defer store.mu.RUnlock()

	count := 0
	for _, session := range store.sessions {
		if session.Registry == nil {
			continue
		}
		count += len(session.Registry.GetByState(state))
	}

	return count
}

/*
const (
	SessionInitialized SessionStateType = iota
	SessionStarting
	SessionRunning
	SessionStopping
	SessionStopped
	SessionFinished
	SessionError
)
*/
// IsValidSessionStateTransition validates if a state transition is allowed
func IsValidSessionStateTransition(from, to SessionStateType) bool {
	validTransitions := map[SessionStateType][]SessionStateType{
		SessionInitialized: {SessionStarting, SessionError},
		SessionStarting:    {SessionRunning, SessionError},
		SessionRunning:     {SessionStopping, SessionError},
		SessionStopping:    {SessionStopped, SessionError},
		SessionStopped:     {SessionFinished, SessionStarting, SessionError},
		SessionFinished:    {}, // just drop the session, idk
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
