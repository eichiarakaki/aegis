package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const PriorityAggTrade = 2

type AggTrade struct {
	AggTradeID   int64   `json:"agg_trade_id"`
	Price        float64 `json:"price"`
	Quantity     float64 `json:"quantity"`
	FirstTradeID int64   `json:"first_trade_id"`
	LastTradeID  int64   `json:"last_trade_id"`
	TransactTime int64   `json:"transact_time"`
	IsBuyerMaker bool    `json:"is_buyer_maker"`
}

// ParseAggTrade parses a raw CSV row into an AggTrade.
// Expected columns:
//
//	0  agg_trade_id
//	1  price
//	2  quantity
//	3  first_trade_id
//	4  last_trade_id
//	5  transact_time
//	6  is_buyer_maker
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
