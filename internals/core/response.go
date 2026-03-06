package core

import (
	"encoding/json"
	"net"
	"time"
)

// Response is the standard reply sent from the daemon to the CLI.
// Data is omitted from JSON when nil so the client never receives "data": null
// or "data": {}. Callers should leave Data as nil when there is nothing
// meaningful to return.
type Response struct {
	RequestID string         `json:"request_id"`
	Command   CLICommandType `json:"command"`
	Status    ForeignType    `json:"status"`
	ErrorCode ErrorCode      `json:"error_code,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      any            `json:"data,omitempty"`
}

// WriteJSON encodes resp to conn. Empty maps and empty slices assigned to Data
// are normalized to nil so they are omitted from the output.
func WriteJSON(conn net.Conn, resp Response) {
	resp.Data = normalizeData(resp.Data)
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		return
	}
}

// normalizeData returns nil for values that would produce meaningless JSON:
//   - nil / untyped nil
//   - empty map[string]any  →  {}
//   - empty []any           →  []  (only top-level; inner arrays are kept)
func normalizeData(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		if len(val) == 0 {
			return nil
		}
	case []any:
		if len(val) == 0 {
			return nil
		}
	}
	return v
}

// ── Session response data ─────────────────────────────────────────────────────

type SessionCreateData struct {
	Session SessionSummary `json:"session"`
}

type SessionAttachData struct {
	SessionID          string         `json:"session_id"`
	AttachedComponents []ComponentRef `json:"attached_components"`
}

type SessionStartData struct {
	SessionID     string         `json:"session_id"`
	PreviousState string         `json:"previous_state"`
	CurrentState  string         `json:"current_state"`
	StartedAt     time.Time      `json:"started_at"`
	Components    []ComponentRef `json:"components"`
}

type SessionStopData struct {
	SessionID     string         `json:"session_id"`
	PreviousState string         `json:"previous_state"`
	CurrentState  string         `json:"current_state"`
	StoppedAt     *time.Time     `json:"stopped_at"`
	Components    []ComponentRef `json:"components"`
}

type SessionListData struct {
	Sessions []SessionListEntry `json:"sessions"`
}

type SessionStateData struct {
	Session    SessionDetail  `json:"session"`
	Components []ComponentRef `json:"components"`
}

type SessionDeleteData struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name"`
}

// ── Component response data ───────────────────────────────────────────────────

type ComponentListData struct {
	SessionID  string             `json:"session_id"`
	Components []ComponentSummary `json:"components"`
}

type ComponentGetData struct {
	SessionID string          `json:"session_id"`
	Component ComponentDetail `json:"component"`
}

type ComponentDescribeData struct {
	SessionID string              `json:"session_id"`
	Component ComponentFullDetail `json:"component"`
}

// ── Shared sub-types ──────────────────────────────────────────────────────────

// ComponentRef is the minimal representation used inside session responses.
type ComponentRef struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// SessionSummary is returned by SESSION_CREATE.
type SessionSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Mode      string    `json:"mode"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionListEntry is one row in SESSION_LIST.
type SessionListEntry struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Mode           string     `json:"mode"`
	State          string     `json:"state"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at"`
	StoppedAt      *time.Time `json:"stopped_at"`
	ComponentCount int        `json:"component_count"`
}

// SessionDetail is returned by SESSION_STATE.
type SessionDetail struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Mode          string     `json:"mode"`
	State         string     `json:"state"`
	UptimeSeconds int64      `json:"uptime_seconds"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at"`
	StoppedAt     *time.Time `json:"stopped_at"`
}

// ComponentSummary is one row in COMPONENT_LIST.
type ComponentSummary struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	State               string          `json:"state"`
	Requires            map[string]bool `json:"requires"`
	SupportedSymbols    []string        `json:"supported_symbols"`
	SupportedTimeframes []string        `json:"supported_timeframes"`
}

// ComponentDetail is returned by COMPONENT_GET.
type ComponentDetail struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	State               string          `json:"state"`
	Requires            map[string]bool `json:"requires"`
	SupportedSymbols    []string        `json:"supported_symbols"`
	SupportedTimeframes []string        `json:"supported_timeframes"`
	StartedAt           time.Time       `json:"started_at"`
	UptimeSeconds       int64           `json:"uptime_seconds"`
}

// ComponentFullDetail is returned by COMPONENT_DESCRIBE.
type ComponentFullDetail struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	State            string           `json:"state"`
	TopicsSubscribed []string         `json:"topics_subscribed"`
	TopicsPublished  []string         `json:"topics_published"`
	Socket           string           `json:"socket,omitempty"`
	Requires         map[string]bool  `json:"requires"`
	Metrics          ComponentMetrics `json:"metrics"`
}

// ComponentMetrics is embedded in ComponentFullDetail.
type ComponentMetrics struct {
	MessagesIn    int64     `json:"messages_in,omitempty"`
	MessagesOut   int64     `json:"messages_out,omitempty"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}
