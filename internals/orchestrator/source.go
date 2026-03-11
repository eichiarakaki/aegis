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
// "orderBook" is absent because it has no CSV representation — it is realtime only.
var parserRegistry = map[string]struct {
	Parse    ParseFunc
	Priority int
}{
	"klines":    {schema.ParseKline, schema.PriorityKline},
	"aggTrades": {schema.ParseAggTrade, schema.PriorityAggTrade},
	"trades":    {schema.ParseTrade, schema.PriorityTrade},
	"bookDepth": {schema.ParseBookDepth, schema.PriorityBookDepth},
	"metrics":   {schema.ParseMetrics, schema.PriorityMetrics},
	// "orderBook" intentionally absent — realtime only, no CSV parser.
}

// realtimePriorityRegistry holds priority values for data types that exist
// only in realtime mode (no CSV equivalent).
var realtimePriorityRegistry = map[string]int{
	"orderBook": schema.PriorityOrderBook,
}

// DataTypeInfo returns the ParseFunc and priority for a given data type.
// For realtime-only types (e.g. "orderBook") it returns (nil, priority, nil) —
// callers must check whether ParseFunc is nil before using it for CSV parsing.
func DataTypeInfo(dataType string) (ParseFunc, int, error) {
	if entry, ok := parserRegistry[dataType]; ok {
		return entry.Parse, entry.Priority, nil
	}
	if priority, ok := realtimePriorityRegistry[dataType]; ok {
		return nil, priority, nil
	}
	return nil, 0, fmt.Errorf("unknown data type: %q", dataType)
}

// RawRow is a fully parsed, normalized row ready to be published.
type RawRow struct {
	Timestamp int64
	DataType  string
	Priority  int
	Topic     string // full NATS topic: aegis.<sid>.<type>.<sym>[.<tf>]
	Payload   []byte // JSON-encoded typed struct
}

// DataSource is the common interface for both CSV and live feed sources.
type DataSource interface {
	Peek() (int64, error)
	Drain(ts int64) ([]RawRow, error)
	Topic() string
	DataType() string
}

// ErrExhausted is returned by Peek/Drain when the source has no more data.
var ErrExhausted = io.EOF
