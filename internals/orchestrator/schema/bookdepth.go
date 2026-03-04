package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const PriorityBookDepth = 4

const bookDepthTimeLayout = "2006-01-02 15:04:05"

type BookDepth struct {
	Timestamp  int64   `json:"timestamp"`
	Percentage float64 `json:"percentage"`
	Depth      float64 `json:"depth"`
	Notional   float64 `json:"notional"`
}

// ParseBookDepth parses a raw CSV row into a BookDepth.
// Expected columns:
//
//	0  timestamp  ("2006-01-02 15:04:05")
//	1  percentage
//	2  depth
//	3  notional
func ParseBookDepth(row []string) (int64, []byte, error) {
	if len(row) < 4 {
		return 0, nil, fmt.Errorf("bookDepth: expected >=4 columns, got %d", len(row))
	}

	t, err := time.Parse(bookDepthTimeLayout, row[0])
	if err != nil {
		return 0, nil, fmt.Errorf("bookDepth: timestamp: %w", err)
	}
	tsMs := t.UnixMilli()

	percentage, err := strconv.ParseFloat(row[1], 64)
	if err != nil {
		return 0, nil, fmt.Errorf("bookDepth: percentage: %w", err)
	}
	depth, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		return 0, nil, fmt.Errorf("bookDepth: depth: %w", err)
	}
	notional, err := strconv.ParseFloat(row[3], 64)
	if err != nil {
		return 0, nil, fmt.Errorf("bookDepth: notional: %w", err)
	}

	b := BookDepth{
		Timestamp:  tsMs,
		Percentage: percentage,
		Depth:      depth,
		Notional:   notional,
	}

	payload, err := json.Marshal(b)
	if err != nil {
		return 0, nil, fmt.Errorf("bookDepth: marshal: %w", err)
	}

	return tsMs, payload, nil
}
