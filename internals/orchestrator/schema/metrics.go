package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const PriorityMetrics = 5

const metricsTimeLayout = "2006-01-02 15:04:05"

type Metrics struct {
	CreateTime                   int64   `json:"create_time"`
	Symbol                       string  `json:"symbol"`
	SumOpenInterest              float64 `json:"sum_open_interest"`
	SumOpenInterestValue         float64 `json:"sum_open_interest_value"`
	CountTopTraderLongShortRatio float64 `json:"count_toptrader_long_short_ratio"`
	SumTopTraderLongShortRatio   float64 `json:"sum_toptrader_long_short_ratio"`
	CountLongShortRatio          float64 `json:"count_long_short_ratio"`
	SumTakerLongShortVolRatio    float64 `json:"sum_taker_long_short_vol_ratio"`
}

// ParseMetrics parses a raw CSV row into a Metrics.
// Expected columns:
//
//	0  create_time  ("2006-01-02 15:04:05")
//	1  symbol
//	2  sum_open_interest
//	3  sum_open_interest_value
//	4  count_toptrader_long_short_ratio
//	5  sum_toptrader_long_short_ratio
//	6  count_long_short_ratio
//	7  sum_taker_long_short_vol_ratio
func ParseMetrics(row []string) (int64, []byte, error) {
	if len(row) < 8 {
		return 0, nil, fmt.Errorf("metrics: expected >=8 columns, got %d", len(row))
	}

	t, err := time.Parse(metricsTimeLayout, row[0])
	if err != nil {
		return 0, nil, fmt.Errorf("metrics: create_time: %w", err)
	}
	tsMs := t.UnixMilli()

	parseF := func(s, field string) (float64, error) {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("metrics: %s: %w", field, err)
		}
		return v, nil
	}

	sumOI, err := parseF(row[2], "sum_open_interest")
	if err != nil {
		return 0, nil, err
	}
	sumOIVal, err := parseF(row[3], "sum_open_interest_value")
	if err != nil {
		return 0, nil, err
	}
	cntTopLS, err := parseF(row[4], "count_toptrader_long_short_ratio")
	if err != nil {
		return 0, nil, err
	}
	sumTopLS, err := parseF(row[5], "sum_toptrader_long_short_ratio")
	if err != nil {
		return 0, nil, err
	}
	cntLS, err := parseF(row[6], "count_long_short_ratio")
	if err != nil {
		return 0, nil, err
	}
	sumTakerLS, err := parseF(row[7], "sum_taker_long_short_vol_ratio")
	if err != nil {
		return 0, nil, err
	}

	m := Metrics{
		CreateTime:                   tsMs,
		Symbol:                       row[1],
		SumOpenInterest:              sumOI,
		SumOpenInterestValue:         sumOIVal,
		CountTopTraderLongShortRatio: cntTopLS,
		SumTopTraderLongShortRatio:   sumTopLS,
		CountLongShortRatio:          cntLS,
		SumTakerLongShortVolRatio:    sumTakerLS,
	}

	payload, err := json.Marshal(m)
	if err != nil {
		return 0, nil, fmt.Errorf("metrics: marshal: %w", err)
	}

	return tsMs, payload, nil
}
