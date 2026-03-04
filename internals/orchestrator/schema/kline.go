package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const PriorityKline = 3

type Kline struct {
	OpenTime            int64   `json:"open_time"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume"`
	CloseTime           int64   `json:"close_time"`
	QuoteVolume         float64 `json:"quote_volume"`
	Count               int64   `json:"count"`
	TakerBuyVolume      float64 `json:"taker_buy_volume"`
	TakerBuyQuoteVolume float64 `json:"taker_buy_quote_volume"`
}

// ParseKline parses a raw CSV row into a Kline.
// Expected columns (index):
//
//	0  open_time
//	1  open
//	2  high
//	3  low
//	4  close
//	5  volume
//	6  close_time
//	7  quote_volume
//	8  count
//	9  taker_buy_volume
//	10 taker_buy_quote_volume
//	11 ignore
func ParseKline(row []string) (int64, []byte, error) {
	if len(row) < 11 {
		return 0, nil, fmt.Errorf("kline: expected >=11 columns, got %d", len(row))
	}

	openTime, err := strconv.ParseInt(row[0], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("kline: open_time: %w", err)
	}

	parseF := func(s string, field string) (float64, error) {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("kline: %s: %w", field, err)
		}
		return v, nil
	}

	open, err := parseF(row[1], "open")
	if err != nil {
		return 0, nil, err
	}
	high, err := parseF(row[2], "high")
	if err != nil {
		return 0, nil, err
	}
	low, err := parseF(row[3], "low")
	if err != nil {
		return 0, nil, err
	}
	close_, err := parseF(row[4], "close")
	if err != nil {
		return 0, nil, err
	}
	volume, err := parseF(row[5], "volume")
	if err != nil {
		return 0, nil, err
	}
	closeTime, err := strconv.ParseInt(row[6], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("kline: close_time: %w", err)
	}
	quoteVolume, err := parseF(row[7], "quote_volume")
	if err != nil {
		return 0, nil, err
	}
	count, err := strconv.ParseInt(row[8], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("kline: count: %w", err)
	}
	takerBuyVolume, err := parseF(row[9], "taker_buy_volume")
	if err != nil {
		return 0, nil, err
	}
	takerBuyQuoteVolume, err := parseF(row[10], "taker_buy_quote_volume")
	if err != nil {
		return 0, nil, err
	}

	k := Kline{
		OpenTime:            openTime,
		Open:                open,
		High:                high,
		Low:                 low,
		Close:               close_,
		Volume:              volume,
		CloseTime:           closeTime,
		QuoteVolume:         quoteVolume,
		Count:               count,
		TakerBuyVolume:      takerBuyVolume,
		TakerBuyQuoteVolume: takerBuyQuoteVolume,
	}

	payload, err := json.Marshal(k)
	if err != nil {
		return 0, nil, fmt.Errorf("kline: marshal: %w", err)
	}

	return openTime, payload, nil
}
