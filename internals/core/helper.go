package core

import "encoding/json"

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
