package core

import (
	"encoding/json"
	"net"
)

// Response structure to the aegisctl
type Response struct {
	RequestID string         `json:"request_id"`
	Command   CLICommandType `json:"command"`
	Status    ForeignType    `json:"status"`
	ErrorCode ErrorCode      `json:"error_code,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      any            `json:"data"`
}

func WriteJSON(conn net.Conn, resp Response) {
	_ = json.NewEncoder(conn).Encode(resp)
}
