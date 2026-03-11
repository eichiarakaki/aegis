package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const PriorityOrderBook = 4 // same slot as bookDepth — they are mutually exclusive

// PriceLevel represents a single bid or ask level in the order book.
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// OrderBook represents a full partial-depth order book snapshot.
// Produced by the realtime WebSocket parser only — there is no CSV equivalent.
type OrderBook struct {
	LastUpdateID int64        `json:"last_update_id"`
	EventTime    int64        `json:"event_time"` // unix ms; 0 if not present in the stream
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
}

// wsOrderBookEvent is the raw Binance partial-depth stream payload.
//
//	{
//	  "lastUpdateId": 160,
//	  "T": 1650000000000,   // event time (present in futures depth stream)
//	  "E": 1650000000001,   // transaction time (futures only, ignored here)
//	  "bids": [["0.0024","10"], ...],
//	  "asks": [["0.0026","100"], ...]
//	}
type wsOrderBookEvent struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	EventTime    int64      `json:"T"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// ParseOrderBook parses a raw Binance WebSocket partial-depth JSON message
// into an OrderBook and returns (timestamp unix ms, JSON payload, error).
func ParseOrderBook(msg []byte) (int64, []byte, error) {
	var ev wsOrderBookEvent
	if err := json.Unmarshal(msg, &ev); err != nil {
		return 0, nil, fmt.Errorf("orderBook: unmarshal: %w", err)
	}

	bids, err := parseLevels(ev.Bids, "bid")
	if err != nil {
		return 0, nil, err
	}
	asks, err := parseLevels(ev.Asks, "ask")
	if err != nil {
		return 0, nil, err
	}

	// Use EventTime when available (USD-M futures depth stream includes it).
	// Fall back to LastUpdateID as a monotonic proxy for spot streams.
	tsMs := ev.EventTime
	if tsMs == 0 {
		tsMs = ev.LastUpdateID
	}

	ob := OrderBook{
		LastUpdateID: ev.LastUpdateID,
		EventTime:    tsMs,
		Bids:         bids,
		Asks:         asks,
	}

	payload, err := json.Marshal(ob)
	if err != nil {
		return 0, nil, fmt.Errorf("orderBook: marshal: %w", err)
	}
	return tsMs, payload, nil
}

// parseLevels converts raw string pairs into typed PriceLevels.
func parseLevels(raw [][]string, side string) ([]PriceLevel, error) {
	levels := make([]PriceLevel, 0, len(raw))
	for _, pair := range raw {
		if len(pair) < 2 {
			continue
		}
		price, err := strconv.ParseFloat(pair[0], 64)
		if err != nil {
			return nil, fmt.Errorf("orderBook: %s price: %w", side, err)
		}
		qty, err := strconv.ParseFloat(pair[1], 64)
		if err != nil {
			return nil, fmt.Errorf("orderBook: %s quantity: %w", side, err)
		}
		levels = append(levels, PriceLevel{Price: price, Quantity: qty})
	}
	return levels, nil
}
