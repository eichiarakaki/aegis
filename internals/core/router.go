package core

type Command struct {
	RequestID string `json:"request_id"`
	Type      string `json:"type"`
	Payload   string `json:"payload"`
}
