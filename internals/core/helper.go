package core

import (
	"encoding/json"
	"os"
	"strings"
)

// DecodePayload marshals raw (typically map[string]any from cmd.Payload)
// back to JSON and unmarshals it into dst. This avoids the boilerplate
// marshal→unmarshal pair repeated in every handler.
func DecodePayload(raw any, dst any) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// DebugEnabled is set once at startup from AEGIS_LOG_LEVEL=debug.
var DebugEnabled = func() bool {
	return strings.ToLower(os.Getenv("AEGIS_LOG_LEVEL")) == "debug"
}()
