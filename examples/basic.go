package main

import (
	"encoding/json"
	"log"
	"net"
)

type Component struct {
	Name                 string   `json:"component_name"`
	Requires             []string `json:"requires"`
	Supported_symbols    []string `json:"supported_symbols"`
	Supported_timeframes []string `json:"supported_timeframes"`
}

func main() {
	socketPath := "/tmp/aegis-components.sock"

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatal("Failed to connect to components socket:", err)
	}
	defer conn.Close()

	component := Component{
		Name:     "basic-strategy",
		Requires: []string{"klines", "orderbook"},
		Supported_symbols: []string{
			"BTCUSDT",
			"ETHUSDT",
		},
		Supported_timeframes: []string{
			"1m",
			"5m",
			"1h",
		},
	}

	err = json.NewEncoder(conn).Encode(component)
	if err != nil {
		log.Fatal("Failed to send component:", err)
		return
	}

	log.Println("Component registered successfully")
}
