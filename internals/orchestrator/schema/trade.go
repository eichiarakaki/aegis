package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const PriorityTrade = 1

type Trade struct {
	ID           int64   `json:"id"`
	Price        float64 `json:"price"`
	Qty          float64 `json:"qty"`
	QuoteQty     float64 `json:"quote_qty"`
	Time         int64   `json:"time"`
	IsBuyerMaker bool    `json:"is_buyer_maker"`
}

// ParseTrade parses a raw CSV row into a Trade.
// Expected columns:
//
//	0  id
//	1  price
//	2  qty
//	3  quote_qty
//	4  time
//	5  is_buyer_maker
func ParseTrade(row []string) (int64, []byte, error) {
	if len(row) < 6 {
		return 0, nil, fmt.Errorf("trade: expected >=6 columns, got %d", len(row))
	}

	id, err := strconv.ParseInt(row[0], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("trade: id: %w", err)
	}
	price, err := strconv.ParseFloat(row[1], 64)
	if err != nil {
		return 0, nil, fmt.Errorf("trade: price: %w", err)
	}
	qty, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		return 0, nil, fmt.Errorf("trade: qty: %w", err)
	}
	quoteQty, err := strconv.ParseFloat(row[3], 64)
	if err != nil {
		return 0, nil, fmt.Errorf("trade: quote_qty: %w", err)
	}
	t, err := strconv.ParseInt(row[4], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("trade: time: %w", err)
	}
	isBuyerMaker := row[5] == "true"

	tr := Trade{
		ID:           id,
		Price:        price,
		Qty:          qty,
		QuoteQty:     quoteQty,
		Time:         t,
		IsBuyerMaker: isBuyerMaker,
	}

	payload, err := json.Marshal(tr)
	if err != nil {
		return 0, nil, fmt.Errorf("trade: marshal: %w", err)
	}

	return t, payload, nil
}
