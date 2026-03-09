package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/logger"
)

type Session struct {
	ID       string
	Name     string
	Mode     string // realtime | historical
	State    SessionStateType
	Registry *Registry

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

	ComponentPaths   []string
	ComponentEntries []ComponentPathEntry
	mu               sync.RWMutex
}

// ComponentPathEntry pairs a binary path with its pre-assigned component ID.
type ComponentPathEntry struct {
	Path        string
	ComponentID string
}

// NewSession creates a new session with the given name and mode.
func NewSession(id string, name string, mode string) *Session {
	return &Session{
		ID:               id,
		Name:             name,
		Mode:             mode,
		State:            SessionInitialized,
		Registry:         NewComponentRegistry(),
		StreamSocket:     nil,
		Topics:           nil,
		TopicOwners:      make(map[string][]string),
		ComponentPaths:   nil,
		ComponentEntries: nil,
		CreatedAt:        time.Now(),
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

	existing := make(map[string]struct{})
	if s.Topics != nil {
		for _, t := range *s.Topics {
			existing[t] = struct{}{}
		}
	}

	for _, t := range newTopics {
		s.TopicOwners[t] = appendUnique(s.TopicOwners[t], componentID)

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

// AddComponentIDForPath stores a path→componentID mapping.
func (s *Session) AddComponentIDForPath(path, componentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, entry := range s.ComponentEntries {
		if entry.Path == path {
			s.ComponentEntries[i].ComponentID = componentID
			return
		}
	}
	s.ComponentEntries = append(s.ComponentEntries, ComponentPathEntry{
		Path:        path,
		ComponentID: componentID,
	})

	for _, p := range s.ComponentPaths {
		if p == path {
			return
		}
	}
	s.ComponentPaths = append(s.ComponentPaths, path)
}

// GetComponentEntries returns a snapshot of path→id entries.
func (s *Session) GetComponentEntries() []ComponentPathEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ComponentPathEntry, len(s.ComponentEntries))
	copy(out, s.ComponentEntries)
	return out
}

// RemoveComponentTopics removes componentID as an owner of its topics.
func (s *Session) RemoveComponentTopics(componentID string, componentTopics []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.TopicOwners == nil || s.Topics == nil {
		return
	}

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

	filtered := (*s.Topics)[:0]
	for _, t := range *s.Topics {
		if _, drop := orphaned[t]; !drop {
			filtered = append(filtered, t)
		}
	}
	*s.Topics = filtered
}

// GetUptimeSeconds calculates and returns the uptime in seconds.
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

func (s *Session) GetStreamSocketPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.StreamSocket != nil {
		return *s.StreamSocket
	}
	return ""
}

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

func (s *Session) SetToStopped() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !IsValidSessionStateTransition(s.State, SessionStopped) {
		return errors.New("session cannot transition to stopped from current state")
	}
	now := time.Now()
	s.State = SessionStopped
	s.StoppedAt = &now
	return nil
}

func (s *Session) SetToFinished() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !IsValidSessionStateTransition(s.State, SessionFinished) {
		return errors.New("session cannot transition to finished from current state")
	}
	now := time.Now()
	s.State = SessionFinished
	s.StoppedAt = &now
	return nil
}

func (s *Session) SetToError() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !IsValidSessionStateTransition(s.State, SessionError) {
		return errors.New("invalid transition to error state")
	}
	s.State = SessionError
	return nil
}

func (s *Session) AddComponent(c *Component) error {
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

func (s *Session) GetState() SessionStateType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

func (s *Session) GetTopicOwnersSnapshot() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TopicOwners == nil {
		return make(map[string][]string)
	}
	copyMap := make(map[string][]string, len(s.TopicOwners))
	for topic, owners := range s.TopicOwners {
		ownersCopy := make([]string, len(owners))
		copy(ownersCopy, owners)
		copyMap[topic] = ownersCopy
	}
	return copyMap
}

func (s *Session) RLock()   { s.mu.RLock() }
func (s *Session) RUnlock() { s.mu.RUnlock() }
func (s *Session) WithRLock(fn func()) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn()
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
	return &SessionStore{sessions: make(map[string]*Session)}
}

func (store *SessionStore) AddSession(s *Session) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if _, exists := store.sessions[s.ID]; exists {
		return fmt.Errorf("session with ID %s already exists", s.ID)
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
	for _, s := range store.sessions {
		if s.Name == name {
			sessions = append(sessions, s)
		}
	}
	return sessions, len(sessions)
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
		if session.Registry != nil {
			count += len(session.Registry.List())
		}
	}
	return count
}

func (store *SessionStore) TotalComponentsByStateFromAllSessions(state ForeignComponentState) int {
	store.mu.RLock()
	defer store.mu.RUnlock()
	count := 0
	for _, session := range store.sessions {
		if session.Registry != nil {
			count += len(session.Registry.GetByState(state))
		}
	}
	return count
}

// GetByHint resolves a session from a token that may be either the session ID
// itself or the session token (hint).
func (store *SessionStore) GetByHint(hint string) (*Session, error) {
	if hint == "" {
		logger.Warn("GetSessionByHint: empty hint provided")
		return nil, fmt.Errorf("empty hint provided")
	}

	logger.WithComponent("sessions").Debugf("Resolving session hint: %s", hint)

	// Try full ID first (highest priority, most specific)
	if session, found := store.GetSessionByID(hint); found {
		logger.WithComponent("sessions").Debugf("Session resolved by full ID: %s", session.ID)
		return session, nil
	}

	// Try ID approximation (first N characters)
	if session, found := store.GetSessionByIDApproximation(hint); found {
		logger.WithComponent("sessions").Debugf("Session resolved by ID approximation: %s", session.ID)
		return session, nil
	}

	// Try name (lowest priority, may have collisions)
	sessions, count := store.GetSessionsByName(hint)
	if count > 1 {
		logger.WithComponent("sessions").Warnf("Multiple sessions found with name '%s' (%d matches)", hint, count)
		return nil, fmt.Errorf("multiple sessions found with name '%s' (%d matches). Use session ID for disambiguation", hint, count)
	}

	if count == 1 && sessions != nil && len(sessions) > 0 {
		logger.WithComponent("sessions").Debugf("Session resolved by name: %s", sessions[0].Name)
		return sessions[0], nil
	}

	logger.WithComponent("sessions").Warnf("Session not found: %s", hint)
	return nil, fmt.Errorf("session not found: %s", hint)
}

// ResetToInitialized resets a FINISHED session back to INITIALIZED so it can
// be restarted. This is only valid from FINISHED state.
// Add this method to the Session struct in session.go.
func (s *Session) ResetToInitialized() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != SessionFinished {
		return fmt.Errorf("ResetToInitialized: invalid from state %s (must be FINISHED)", s.State)
	}

	s.State = SessionInitialized
	s.StartedAt = nil
	s.StoppedAt = nil
	return nil
}

// ForceState sets the session state unconditionally. Only use for rollback.
func (s *Session) ForceState(state SessionStateType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
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
