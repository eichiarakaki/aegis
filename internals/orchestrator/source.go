package orchestrator

import (
	"fmt"
	"io"

	"github.com/eichiarakaki/aegis/internals/orchestrator/schema"
)

// ParseFunc parses a raw CSV row and returns the canonical timestamp (unix ms)
// and the JSON-encoded payload.
type ParseFunc func(row []string) (ts int64, payload []byte, err error)

// parserRegistry maps data_type → ParseFunc and priority.
var parserRegistry = map[string]struct {
	Parse    ParseFunc
	Priority int
}{
	"klines":    {schema.ParseKline, schema.PriorityKline},
	"aggTrades": {schema.ParseAggTrade, schema.PriorityAggTrade},
	"trades":    {schema.ParseTrade, schema.PriorityTrade},
	"bookDepth": {schema.ParseBookDepth, schema.PriorityBookDepth},
	"metrics":   {schema.ParseMetrics, schema.PriorityMetrics},
}

// DataTypeInfo returns the ParseFunc and priority for a given data type.
// Returns an error if the data type is unknown.
func DataTypeInfo(dataType string) (ParseFunc, int, error) {
	entry, ok := parserRegistry[dataType]
	if !ok {
		return nil, 0, fmt.Errorf("unknown data type: %q", dataType)
	}
	return entry.Parse, entry.Priority, nil
}

// RawRow is a fully parsed, normalized row ready to be published.
type RawRow struct {
	Timestamp int64 // canonical unix ms
	DataType  string
	Priority  int
	Topic     string // full NATS topic: aegis.<sid>.<type>.<sym>[.<tf>]
	Payload   []byte // JSON-encoded typed struct
}

// DataSource is the common interface for both CSV and live feed sources.
// Each DataSource represents a single (symbol, data_type[, timeframe]) stream.
type DataSource interface {
	// Peek returns the timestamp of the next available row without consuming it.
	// Returns (0, io.EOF) when the source is fully exhausted.
	Peek() (int64, error)

	// Drain consumes and returns all rows whose timestamp equals ts.
	// If the next row has a different timestamp, returns an empty slice and nil.
	Drain(ts int64) ([]RawRow, error)

	// Topic returns the full NATS topic string for this source.
	Topic() string

	// DataType returns the data type string ("klines", "trades", etc.)
	DataType() string
}

// Sentinel re-export so callers don't import io directly.
var ErrExhausted = io.EOF
