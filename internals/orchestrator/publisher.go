package orchestrator

import (
	"encoding/json"
	"fmt"

	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/nats-io/nats.go"
)

// Envelope is the NATS message wrapper published for every row.
type Envelope struct {
	SessionID string          `json:"session_id"`
	Topic     string          `json:"topic"`
	Ts        int64           `json:"ts"`
	Data      json.RawMessage `json:"data"`
}

// Publisher wraps a NATS connection and publishes Envelopes.
type Publisher struct {
	nc  *nats.Conn
	log *logger.Logger
}

// NewPublisher creates a Publisher from an existing NATS connection.
func NewPublisher(nc *nats.Conn) *Publisher {
	return &Publisher{
		nc:  nc,
		log: logger.WithComponent("Publisher"),
	}
}

// Publish builds an Envelope from a RawRow and publishes it to NATS.
// On first failure it retries once; on second failure it returns the error.
func (p *Publisher) Publish(sessionID string, row RawRow) error {
	env := Envelope{
		SessionID: sessionID,
		Topic:     row.Topic,
		Ts:        row.Timestamp,
		Data:      json.RawMessage(row.Payload),
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("publisher: marshal envelope: %w", err)
	}

	p.log.Debugf("→ %s ts=%d bytes=%d", row.Topic, row.Timestamp, len(data))

	if err := p.nc.Publish(row.Topic, data); err != nil {
		// Retry once.
		if err2 := p.nc.Publish(row.Topic, data); err2 != nil {
			return fmt.Errorf("publisher: publish to %q (retry): %w", row.Topic, err2)
		}
	}
	return nil
}
