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

// OrderBook represents a partial-depth order book snapshot.
// Produced by the realtime WebSocket parser only — there is no CSV equivalent.
type OrderBook struct {
	LastUpdateID int64        `json:"last_update_id"`
	EventTime    int64        `json:"event_time"` // unix ms; 0 if not present in the stream
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
}

// wsOrderBookEvent handles both the partial-depth snapshot format and the
// diff depth stream format that Binance futures uses for @depth20@Xms streams.
//
// Partial-depth snapshot (@depth<N> without speed suffix):
//
//	{ "lastUpdateId": 160, "T": 1650000000000, "bids": [...], "asks": [...] }
//
// Diff depth stream (@depth20@100ms on futures):
//
//	{ "e": "depthUpdate", "E": 123456789, "T": 123456788, "s": "BTCUSDT",
//	  "U": 157, "u": 160, "pu": 149, "b": [...], "a": [...] }
//
// Both formats are handled by reading whichever bid/ask field is present.
type wsOrderBookEvent struct {
	// Snapshot format fields
	LastUpdateID int64      `json:"lastUpdateId"`
	BidsSnapshot [][]string `json:"bids"`
	AsksSnapshot [][]string `json:"asks"`

	// Diff format fields
	EventType    string     `json:"e"`
	EventTime    int64      `json:"E"` // event time (diff format)
	TransactTime int64      `json:"T"` // transaction time (both formats)
	FinalUpdateU int64      `json:"u"` // final update ID (diff format)
	BidsDiff     [][]string `json:"b"`
	AsksDiff     [][]string `json:"a"`
}

// ParseOrderBook parses a Binance WebSocket order book message.
// Handles both partial-depth snapshot and diff depth stream formats.
func ParseOrderBook(msg []byte) (int64, []byte, error) {
	var ev wsOrderBookEvent
	if err := json.Unmarshal(msg, &ev); err != nil {
		return 0, nil, fmt.Errorf("orderBook: unmarshal: %w", err)
	}

	// Determine which bid/ask fields to use based on which format arrived.
	// Diff format has "e":"depthUpdate" and uses "b"/"a".
	// Snapshot format uses "bids"/"asks".
	var rawBids, rawAsks [][]string
	var tsMs int64

	if ev.EventType == "depthUpdate" {
		// Diff depth stream format (@depth20@Xms on futures)
		rawBids = ev.BidsDiff
		rawAsks = ev.AsksDiff
		// Use transaction time ("T") as canonical timestamp — matches aggTrade behavior.
		tsMs = ev.TransactTime
		if tsMs == 0 {
			tsMs = ev.EventTime
		}
		// Use final update ID as last_update_id for ordering.
		ev.LastUpdateID = ev.FinalUpdateU
	} else {
		// Partial-depth snapshot format
		rawBids = ev.BidsSnapshot
		rawAsks = ev.AsksSnapshot
		tsMs = ev.TransactTime
		if tsMs == 0 {
			tsMs = ev.LastUpdateID
		}
	}

	bids, err := parseLevels(rawBids, "bid")
	if err != nil {
		return 0, nil, err
	}
	asks, err := parseLevels(rawAsks, "ask")
	if err != nil {
		return 0, nil, err
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
// Levels with quantity == 0 are included — the consumer is responsible for
// removing them from a local order book (standard Binance diff stream semantics).
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
