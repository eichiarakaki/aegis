package orchestrator

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/eichiarakaki/aegis/internals/orchestrator/schema"
)

// WSParseFunc parses a raw Binance WebSocket JSON payload (the "data" field of
// a combined-stream message) into a (timestamp unix ms, JSON payload) pair.
// The payload must use the same schema structs as the CSV parsers so that
// downstream components receive identical structures regardless of source.
type WSParseFunc func(msg []byte) (ts int64, payload []byte, err error)

// wsParserRegistry maps data_type → WSParseFunc.
//
// "bookDepth" is intentionally absent: it is only available in historical
// (CSV) mode. Requesting it in realtime returns a clear error at startup.
// Use "orderBook" for realtime order book data instead.
var wsParserRegistry = map[string]WSParseFunc{
	"klines":    parseWSKline,
	"aggTrades": parseWSAggTrade,
	"trades":    parseWSTrade,
	"orderBook": parseWSOrderBook,
}

// WSParserFor returns the WSParseFunc for a given data type.
func WSParserFor(dataType string) (WSParseFunc, error) {
	if dataType == "bookDepth" {
		return nil, fmt.Errorf(
			"ws_parser: \"bookDepth\" is not available in realtime mode — " +
				"use \"orderBook\" for live order book snapshots",
		)
	}
	fn, ok := wsParserRegistry[dataType]
	if !ok {
		return nil, fmt.Errorf("ws_parser: no WebSocket stream for data type %q", dataType)
	}
	return fn, nil
}

// toFloat64 converts a json.Number to float64.
// json.Number accepts both JSON strings and JSON numbers, making it robust
// against Binance's inconsistent serialization across endpoints.
func toFloat64(n json.Number, field string) (float64, error) {
	v, err := n.Float64()
	if err != nil {
		return 0, fmt.Errorf("ws kline: %s: %w", field, err)
	}
	return v, nil
}

// ── klines ───────────────────────────────────────────────────────────────────
//
// Binance kline stream. OHLCV fields vary by endpoint:
//   - fstream.binance.com (USD-M futures): may be string or number depending
//     on the server version. Using json.Number handles both transparently.
//
// Shape:
//
//	{
//	  "e": "kline", "E": 123456789, "s": "BTCUSDT",
//	  "k": {
//	    "t": 123400000,  "T": 123460000,
//	    "o": "0.001",    "c": "0.002",   "h": "0.003",  "l": "0.001",
//	    "v": "100",      "n": 100,
//	    "x": false,
//	    "q": "1.0",      "V": "50",      "Q": "0.5"
//	  }
//	}

type wsKlineEvent struct {
	K struct {
		T  int64       `json:"t"` // open time (always a number)
		CT int64       `json:"T"` // close time (always a number)
		O  json.Number `json:"o"`
		H  json.Number `json:"h"`
		L  json.Number `json:"l"`
		C  json.Number `json:"c"`
		V  json.Number `json:"v"`
		Q  json.Number `json:"q"` // quote volume
		N  int64       `json:"n"` // trade count (always a number)
		BV json.Number `json:"V"` // taker buy base volume
		BQ json.Number `json:"Q"` // taker buy quote volume
	} `json:"k"`
}

func parseWSKline(msg []byte) (int64, []byte, error) {
	var ev wsKlineEvent
	if err := json.Unmarshal(msg, &ev); err != nil {
		return 0, nil, fmt.Errorf("ws kline: unmarshal: %w", err)
	}
	k := ev.K

	open, err := toFloat64(k.O, "open")
	if err != nil {
		return 0, nil, err
	}
	high, err := toFloat64(k.H, "high")
	if err != nil {
		return 0, nil, err
	}
	low, err := toFloat64(k.L, "low")
	if err != nil {
		return 0, nil, err
	}
	close_, err := toFloat64(k.C, "close")
	if err != nil {
		return 0, nil, err
	}
	volume, err := toFloat64(k.V, "volume")
	if err != nil {
		return 0, nil, err
	}
	quoteVol, err := toFloat64(k.Q, "quote_volume")
	if err != nil {
		return 0, nil, err
	}
	takerBase, err := toFloat64(k.BV, "taker_buy_volume")
	if err != nil {
		return 0, nil, err
	}
	takerQuote, err := toFloat64(k.BQ, "taker_buy_quote_volume")
	if err != nil {
		return 0, nil, err
	}

	out := schema.Kline{
		OpenTime:            k.T,
		Open:                open,
		High:                high,
		Low:                 low,
		Close:               close_,
		Volume:              volume,
		CloseTime:           k.CT,
		QuoteVolume:         quoteVol,
		Count:               k.N,
		TakerBuyVolume:      takerBase,
		TakerBuyQuoteVolume: takerQuote,
	}
	payload, err := json.Marshal(out)
	if err != nil {
		return 0, nil, fmt.Errorf("ws kline: marshal: %w", err)
	}
	return k.T, payload, nil
}

// ── aggTrades ────────────────────────────────────────────────────────────────

type wsAggTradeEvent struct {
	AggTradeID   int64  `json:"a"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	FirstTradeID int64  `json:"f"`
	LastTradeID  int64  `json:"l"`
	TransactTime int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
}

func parseWSAggTrade(msg []byte) (int64, []byte, error) {
	var ev wsAggTradeEvent
	if err := json.Unmarshal(msg, &ev); err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: unmarshal: %w", err)
	}

	price, err := strconv.ParseFloat(ev.Price, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: price: %w", err)
	}
	qty, err := strconv.ParseFloat(ev.Quantity, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: quantity: %w", err)
	}

	out := schema.AggTrade{
		AggTradeID:   ev.AggTradeID,
		Price:        price,
		Quantity:     qty,
		FirstTradeID: ev.FirstTradeID,
		LastTradeID:  ev.LastTradeID,
		TransactTime: ev.TransactTime,
		IsBuyerMaker: ev.IsBuyerMaker,
	}
	payload, err := json.Marshal(out)
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: marshal: %w", err)
	}
	return ev.TransactTime, payload, nil
}

// ── trades ───────────────────────────────────────────────────────────────────

type wsTradeEvent struct {
	ID           int64  `json:"t"`
	Price        string `json:"p"`
	Qty          string `json:"q"`
	Time         int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
}

func parseWSTrade(msg []byte) (int64, []byte, error) {
	var ev wsTradeEvent
	if err := json.Unmarshal(msg, &ev); err != nil {
		return 0, nil, fmt.Errorf("ws trade: unmarshal: %w", err)
	}

	price, err := strconv.ParseFloat(ev.Price, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("ws trade: price: %w", err)
	}
	qty, err := strconv.ParseFloat(ev.Qty, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("ws trade: qty: %w", err)
	}

	out := schema.Trade{
		ID:           ev.ID,
		Price:        price,
		Qty:          qty,
		QuoteQty:     price * qty,
		Time:         ev.Time,
		IsBuyerMaker: ev.IsBuyerMaker,
	}
	payload, err := json.Marshal(out)
	if err != nil {
		return 0, nil, fmt.Errorf("ws trade: marshal: %w", err)
	}
	return ev.Time, payload, nil
}

// ── orderBook ────────────────────────────────────────────────────────────────

func parseWSOrderBook(msg []byte) (int64, []byte, error) {
	return schema.ParseOrderBook(msg)
}
