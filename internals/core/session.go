package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/orchestrator"
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

	StreamSocket *string
	Topics       *[]string

	// TopicOwners tracks which components declared each topic.
	// topic -> []componentID
	// When a component unregisters, its topics are removed only if no
	// other component still owns them.
	TopicOwners map[string][]string

	Orchestrator *orchestrator.Orchestrator

	ComponentPaths []string

	mu sync.RWMutex
}

// NewSession creates a new session with the given name and mode.
func NewSession(id string, name string, mode string) *Session {
	return &Session{
		ID:             id,
		Name:           name,
		Mode:           mode,
		State:          SessionInitialized,
		Registry:       component.NewComponentRegistry(),
		StreamSocket:   nil,
		Topics:         nil,
		TopicOwners:    make(map[string][]string),
		ComponentPaths: nil,
		Orchestrator:   nil,
		CreatedAt:      time.Now(),
	}
}

// AddTopics merges newTopics into the session topic list and records componentID
// as an owner of each topic. Topics already present are not duplicated.
func (s *Session) AddTopics(componentID string, newTopics []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.TopicOwners == nil {
		s.TopicOwners = make(map[string][]string)
	}

	// Build a set of existing topics for O(1) lookup.
	existing := make(map[string]struct{})
	if s.Topics != nil {
		for _, t := range *s.Topics {
			existing[t] = struct{}{}
		}
	}

	for _, t := range newTopics {
		// Register the owner regardless — the component may reconnect.
		s.TopicOwners[t] = appendUnique(s.TopicOwners[t], componentID)

		// Only append to the flat list if not already there.
		if _, ok := existing[t]; !ok {
			existing[t] = struct{}{}
			if s.Topics == nil {
				topics := []string{t}
				s.Topics = &topics
			} else {
				*s.Topics = append(*s.Topics, t)
			}
		}
	}
}

// AddComponentPath appends a binary path if not already present.
func (s *Session) AddComponentPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.ComponentPaths {
		if p == path {
			return
		}
	}
	s.ComponentPaths = append(s.ComponentPaths, path)
}

// GetComponentPaths returns a snapshot of the stored binary paths.
func (s *Session) GetComponentPaths() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.ComponentPaths))
	copy(out, s.ComponentPaths)
	return out
}

// RemoveComponentTopics removes componentID as an owner of its topics.
// A topic is removed from the session only when it has no remaining owners.
func (s *Session) RemoveComponentTopics(componentID string, componentTopics []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.TopicOwners == nil || s.Topics == nil {
		return
	}

	// Determine which topics become orphaned after removing this owner.
	orphaned := make(map[string]struct{})
	for _, t := range componentTopics {
		owners := removeValue(s.TopicOwners[t], componentID)
		if len(owners) == 0 {
			delete(s.TopicOwners, t)
			orphaned[t] = struct{}{}
		} else {
			s.TopicOwners[t] = owners
		}
	}

	if len(orphaned) == 0 {
		return
	}

	// Rebuild the flat topic list without orphaned topics.
	filtered := (*s.Topics)[:0]
	for _, t := range *s.Topics {
		if _, drop := orphaned[t]; !drop {
			filtered = append(filtered, t)
		}
	}
	*s.Topics = filtered
}

// GetUptimeSeconds calculates and returns the uptime in seconds.
// Returns 0 if the session has not started yet.
// If the session is running or was stopped, returns the duration between StartedAt and StoppedAt (or now).
func (s *Session) GetUptimeSeconds() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.StartedAt == nil {
		return 0
	}

	if s.StoppedAt == nil {
		return int64(time.Since(*s.StartedAt).Seconds())
	}

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

// GetStreamSocketPath just returns the StreamSocket.
func (s *Session) GetStreamSocketPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.StreamSocket != nil {
		return *s.StreamSocket
	}
	return ""
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

// SetToFinished transitions the session from SessionStopped to SessionFinished.
func (s *Session) SetToFinished() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !IsValidSessionStateTransition(s.State, SessionFinished) {
		return errors.New("session cannot transition to stopped from current state")
	}

	now := time.Now()
	s.State = SessionFinished
	s.StartedAt = &now
	return nil
}

// SetToError transitions the session from any to SessionError.
func (s *Session) SetToError() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !IsValidSessionStateTransition(s.State, SessionError) {
		return errors.New("session cannot transition to stopped from current state")
	}

	now := time.Now()
	s.State = SessionError
	s.StartedAt = &now
	return nil
}

// AddComponent adds a component to the session.
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

// ---------- helpers ----------

func appendUnique(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

func removeValue(slice []string, value string) []string {
	out := slice[:0]
	for _, v := range slice {
		if v != value {
			out = append(out, v)
		}
	}
	return out
}

// ---------- SessionStore ----------

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

func (store *SessionStore) GetSessionByIDApproximation(approximation string) (*Session, bool) {
	if approximation == "" {
		return nil, false
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	for _, session := range store.sessions {
		if session.ID == approximation {
			return session, true
		}
	}

	maxIDLength := 0
	for _, session := range store.sessions {
		if len(session.ID) > maxIDLength {
			maxIDLength = len(session.ID)
		}
	}

	for prefixLen := len(approximation); prefixLen <= maxIDLength; prefixLen++ {
		if prefixLen > len(approximation) {
			break
		}

		matches := 0
		var matchedSession *Session

		for _, session := range store.sessions {
			if len(session.ID) >= prefixLen && session.ID[:prefixLen] == approximation[:prefixLen] {
				matches++
				matchedSession = session
				if matches > 1 {
					break
				}
			}
		}

		if matches == 1 {
			return matchedSession, true
		}

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

func IsValidSessionStateTransition(from, to SessionStateType) bool {
	validTransitions := map[SessionStateType][]SessionStateType{
		SessionInitialized: {SessionStarting, SessionStopping, SessionError},
		SessionStarting:    {SessionRunning, SessionError},
		SessionRunning:     {SessionStopping, SessionError},
		SessionStopping:    {SessionStopped, SessionError},
		SessionStopped:     {SessionFinished, SessionStarting, SessionError},
		SessionFinished:    {},
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
