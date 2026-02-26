package components

import (
	"encoding/json"
	"log"
	"net"
)

type Component struct {
	Name                 string   `json:"component_name"`
	Requires             []string `json:"requires"` // klines, orders, etc.
	Supported_symbols    []string `json:"supported_symbols"`
	Supported_timeframes []string `json:"supported_timeframes"`
}

func HandleComponentConnections(conn net.Conn) {
	defer conn.Close()

	var component Component
	err := json.NewDecoder(conn).Decode(&component)
	if err != nil {
		log.Println("Invalid component:", err)
		return
	}

	log.Printf("Received component: %s | Requires: %v | Supported symbols: %v | Supported timeframes: %v\n", component.Name, component.Requires, component.Supported_symbols, component.Supported_timeframes)

}
