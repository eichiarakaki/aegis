package core

import (
	"fmt"
	"strings"
)

type Requires struct {
	Klines               bool `json:"klines"`
	LiquidationSnapshots bool `json:"liquidation_snapshots"`
	Metrics              bool `json:"metrics"`
	AggTrades            bool `json:"agg_trades"`
	BookDepth            bool `json:"book_depth"`
	Trades               bool `json:"trades"`
}

type Component struct {
	Name                 string     `json:"component_name"`
	Requires             []Requires `json:"requires"`
	Supported_symbols    []string   `json:"supported_symbols"`
	Supported_timeframes []string   `json:"supported_timeframes"`
}

const ValidIntervals = "1m,3m,5m,15m,30m,1h,2h,4h,6h,8h,12h,1d,3d,1w,1M"

func (c *Component) ValidateInterval() error {
	for _, interval := range c.Supported_timeframes {
		if !strings.Contains(ValidIntervals, interval) {
			return fmt.Errorf("invalid interval: %s", interval)
		}
	}
	return nil
}
