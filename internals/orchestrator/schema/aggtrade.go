package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const PriorityAggTrade = 2

// AggTrade matches the Binance aggTrade WebSocket stream schema exactly.
//
// Realtime fields (from WebSocket):
//   - all fields populated
//
// Historical fields (from Binance Vision CSV):
//   - EventTime and Symbol are zero/empty — not present in CSV files
//   - NormalQty is zero — not present in CSV files
//
// Consumers should check EventTime == 0 to detect historical rows and
// fall back to TransactTime as the canonical timestamp in both modes.
type AggTrade struct {
	EventTime    int64   `json:"event_time"`     // "E" — zero in historical mode
	Symbol       string  `json:"symbol"`         // "s" — empty in historical mode
	AggTradeID   int64   `json:"agg_trade_id"`   // "a"
	Price        float64 `json:"price"`          // "p"
	Quantity     float64 `json:"quantity"`       // "q" — total qty including RPI orders
	NormalQty    float64 `json:"normal_qty"`     // "nq" — qty excluding RPI orders; zero in historical mode
	FirstTradeID int64   `json:"first_trade_id"` // "f"
	LastTradeID  int64   `json:"last_trade_id"`  // "l"
	TransactTime int64   `json:"transact_time"`  // "T" — canonical timestamp in both modes
	IsBuyerMaker bool    `json:"is_buyer_maker"` // "m"
}

// ParseAggTrade parses a raw Binance Vision CSV row into an AggTrade.
//
// Expected columns (Binance Vision format):
//
//	0  agg_trade_id
//	1  price
//	2  quantity
//	3  first_trade_id
//	4  last_trade_id
//	5  transact_time
//	6  is_buyer_maker
//
// EventTime, Symbol, and NormalQty are not present in CSV files and are
// left at their zero values. Consumers must handle this gracefully.
func ParseAggTrade(row []string) (int64, []byte, error) {
	if len(row) < 7 {
		return 0, nil, fmt.Errorf("aggTrade: expected >=7 columns, got %d", len(row))
	}

	aggTradeID, err := strconv.ParseInt(row[0], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("aggTrade: agg_trade_id: %w", err)
	}
	price, err := strconv.ParseFloat(row[1], 64)
	if err != nil {
		return 0, nil, fmt.Errorf("aggTrade: price: %w", err)
	}
	quantity, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		return 0, nil, fmt.Errorf("aggTrade: quantity: %w", err)
	}
	firstTradeID, err := strconv.ParseInt(row[3], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("aggTrade: first_trade_id: %w", err)
	}
	lastTradeID, err := strconv.ParseInt(row[4], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("aggTrade: last_trade_id: %w", err)
	}
	transactTime, err := strconv.ParseInt(row[5], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("aggTrade: transact_time: %w", err)
	}
	isBuyerMaker := row[6] == "true"

	a := AggTrade{
		// EventTime, Symbol, NormalQty left as zero — not in CSV
		AggTradeID:   aggTradeID,
		Price:        price,
		Quantity:     quantity,
		FirstTradeID: firstTradeID,
		LastTradeID:  lastTradeID,
		TransactTime: transactTime,
		IsBuyerMaker: isBuyerMaker,
	}

	payload, err := json.Marshal(a)
	if err != nil {
		return 0, nil, fmt.Errorf("aggTrade: marshal: %w", err)
	}

	return transactTime, payload, nil
}
