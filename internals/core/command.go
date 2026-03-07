package core

type ComponentLogPathPayload struct {
	SessionID   string `json:"session_id"`
	ComponentID string `json:"component_id"`
}

// SessionStartPayload is sent by the CLI when starting a session.
// From and To are optional unix millisecond timestamps for historical range filtering.
// If zero, the full dataset is used.
type SessionStartPayload struct {
	SessionID string `json:"session_id"`
	From      int64  `json:"from,omitempty"` // unix ms, inclusive
	To        int64  `json:"to,omitempty"`   // unix ms, inclusive
}
