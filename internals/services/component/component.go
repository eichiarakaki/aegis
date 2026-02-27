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

	if err := component.ValidateInterval(); err != nil {
		logger.Warnf("Component %s has invalid interval: %v", component.Name, err)
		return
	}

	for _, req := range component.Requires {
		logger.Debugf("Component %s requires: klines=%t, liquidation_snapshots=%t, metrics=%t, agg_trades=%t, book_depth=%t, trades=%t",
			component.Name, req.Klines, req.LiquidationSnapshots, req.Metrics, req.AggTrades, req.BookDepth, req.Trades)
	}
	logger.Debugf("Component %s supports symbols: %v", component.Name, component.Supported_symbols)
	logger.Debugf("Component %s supports timeframes: %v", component.Name, component.Supported_timeframes)
}
