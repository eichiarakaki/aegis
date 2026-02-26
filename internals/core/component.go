package core

type Component struct {
	Name                 string   `json:"component_name"`
	Requires             []string `json:"requires"` // klines, orders, etc.
	Supported_symbols    []string `json:"supported_symbols"`
	Supported_timeframes []string `json:"supported_timeframes"`
}
