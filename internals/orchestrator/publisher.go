package orchestrator

import (
	"encoding/json"
	"fmt"

	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/nats-io/nats.go"
)

// Envelope is the message wrapper published for every row.
type Envelope struct {
	SessionID string          `json:"session_id"`
	Topic     string          `json:"topic"`
	Ts        int64           `json:"ts"`
	Data      json.RawMessage `json:"data"`
}

// Publisher routes rows either to a DataStreamServer (historical mode with
// backpressure) or directly to NATS (realtime mode).
type Publisher struct {
	nc  *nats.Conn
	ds  *DataStreamServer // nil in realtime-only mode
	log *logger.Logger
}

// NewPublisher creates a Publisher.
// If ds is non-nil, Publish delivers to it synchronously (backpressure).
// If ds is nil, Publish falls back to raw NATS publish (realtime).
func NewPublisher(nc *nats.Conn, ds *DataStreamServer) *Publisher {
	return &Publisher{
		nc:  nc,
		ds:  ds,
		log: logger.WithComponent("Publisher"),
	}
}

// Publish builds an Envelope from a RawRow and delivers it.
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

	if p.ds != nil {
		// Historical mode: block until every interested component receives it.
		p.ds.Deliver(row.Topic, data)
		return nil
	}

	// Realtime mode: publish to NATS, retry once on failure.
	if err := p.nc.Publish(row.Topic, data); err != nil {
		if err2 := p.nc.Publish(row.Topic, data); err2 != nil {
			return fmt.Errorf("publisher: publish to %q (retry): %w", row.Topic, err2)
		}
	}
	return nil
}
