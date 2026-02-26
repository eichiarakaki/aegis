package components

import (
	"encoding/json"
	"log"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
)

func HandleComponentConnections(conn net.Conn) {
	defer conn.Close()

	var component core.Component
	err := json.NewDecoder(conn).Decode(&component)
	if err != nil {
		log.Println("Invalid component:", err)
		return
	}

	logger.Infof("Received component: %s | Requires: %v | Supported symbols: %v | Supported timeframes: %v\n", component.Name, component.Requires, component.Supported_symbols, component.Supported_timeframes)

}
