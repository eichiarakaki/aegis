package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Requires struct {
	Klines               *bool `json:"klines,omitempty"`
	LiquidationSnapshots *bool `json:"liquidation_snapshots,omitempty"`
	Metrics              *bool `json:"metrics,omitempty"`
	AggTrades            *bool `json:"agg_trades,omitempty"`
	BookDepth            *bool `json:"book_depth,omitempty"`
	Trades               *bool `json:"trades,omitempty"`
}

type ComponentStateType int

const (
	ComponentPending ComponentStateType = iota
	ComponentStarting
	ComponentRunning
	ComponentStopping
	ComponentStopped
	ComponentFinished
	ComponentError
)

func ComponentStateToString(state ComponentStateType) string {
	switch state {
	case ComponentPending:
		return "pending"
	case ComponentStarting:
		return "starting"
	case ComponentRunning:
		return "running"
	case ComponentStopping:
		return "stopping"
	case ComponentStopped:
		return "stopped"
	case ComponentFinished:
		return "finished"
	case ComponentError:
		return "error"
	default:
		return "unknown"
	}
}

type Component struct {
	ID                  *string            `json:"id,omitempty"`
	Name                string             `json:"name,omitempty"`
	Requires            *[]Requires        `json:"requires,omitempty"`
	SupportedSymbols    []string           `json:"supported_symbols"`
	SupportedTimeframes []string           `json:"supported_timeframes"`
	State               ComponentStateType `json:"-"`
	Path                string             `json:"path,omitempty"`
}

// MarshalJSON Customized to serialize State as string
func (c Component) MarshalJSON() ([]byte, error) {
	type Alias Component
	return json.Marshal(&struct {
		State string `json:"state"`
		*Alias
	}{
		State: ComponentStateToString(c.State),
		Alias: (*Alias)(&c),
	})
}

const ValidIntervals = "1m,3m,5m,15m,30m,1h,2h,4h,6h,8h,12h,1d,3d,1w,1M"

func (c Component) ValidateInterval() error {
	for _, interval := range c.SupportedTimeframes {
		if !strings.Contains(ValidIntervals, interval) {
			return fmt.Errorf("invalid interval: %s", interval)
		}
	}
	return nil
}
