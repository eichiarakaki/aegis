package core

import (
	"encoding/json"
	"net"
)

type Response struct {
	RequestID string `json:"request_id"`
	Command   string `json:"command"`
	Status    string `json:"status"` // ok | error
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message,omitempty"`
	Data      any    `json:"data"`
}

func WriteJSON(conn net.Conn, resp Response) {
	_ = json.NewEncoder(conn).Encode(resp)
}
