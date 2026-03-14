package orchestrator

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/eichiarakaki/aegis/internals/orchestrator/schema"
)

type WSParseFunc func(msg []byte) (ts int64, payload []byte, err error)

var wsParserRegistry = map[string]WSParseFunc{
	"klines":    parseWSKline,
	"aggTrades": parseWSAggTrade,
	"trades":    parseWSTrade,
	"orderBook": parseWSOrderBook,
}

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

func toFloat64(n json.Number, field string) (float64, error) {
	v, err := n.Float64()
	if err != nil {
		return 0, fmt.Errorf("ws kline: %s: %w", field, err)
	}
	return v, nil
}

// ── klines ───────────────────────────────────────────────────────────────────

type wsKlineEvent struct {
	K struct {
		T  int64       `json:"t"`
		CT int64       `json:"T"`
		O  json.Number `json:"o"`
		H  json.Number `json:"h"`
		L  json.Number `json:"l"`
		C  json.Number `json:"c"`
		V  json.Number `json:"v"`
		Q  json.Number `json:"q"`
		N  int64       `json:"n"`
		BV json.Number `json:"V"`
		BQ json.Number `json:"Q"`
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
//
// We cannot use a struct with tags `json:"e"` and `json:"E"` simultaneously
// because Go's encoding/json decoder is case-insensitive: it matches
// "e":"aggTrade" onto the field tagged `json:"E"` (EventTime), producing
// the error "cannot unmarshal string into Number".
//
// Solution: unmarshal into map[string]json.RawMessage for exact case-sensitive
// key matching, then extract fields manually.

func parseWSAggTrade(msg []byte) (int64, []byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(msg, &raw); err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: unmarshal: %w", err)
	}

	parseInt64 := func(key string) (int64, error) {
		v, ok := raw[key]
		if !ok {
			return 0, nil
		}
		var n json.Number
		if err := json.Unmarshal(v, &n); err != nil {
			return 0, fmt.Errorf("field %q: %w", key, err)
		}
		return n.Int64()
	}

	parseStr := func(key string) string {
		v, ok := raw[key]
		if !ok {
			return ""
		}
		var s string
		json.Unmarshal(v, &s)
		return s
	}

	parseBool := func(key string) bool {
		v, ok := raw[key]
		if !ok {
			return false
		}
		var b bool
		json.Unmarshal(v, &b)
		return b
	}

	eventTime, err := parseInt64("E")
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: event_time: %w", err)
	}
	aggTradeID, err := parseInt64("a")
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: agg_trade_id: %w", err)
	}
	firstTradeID, err := parseInt64("f")
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: first_trade_id: %w", err)
	}
	lastTradeID, err := parseInt64("l")
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: last_trade_id: %w", err)
	}
	transactTime, err := parseInt64("T")
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: transact_time: %w", err)
	}

	price, err := strconv.ParseFloat(parseStr("p"), 64)
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: price: %w", err)
	}
	qty, err := strconv.ParseFloat(parseStr("q"), 64)
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: quantity: %w", err)
	}

	var normalQty float64
	if nqStr := parseStr("nq"); nqStr != "" {
		normalQty, err = strconv.ParseFloat(nqStr, 64)
		if err != nil {
			return 0, nil, fmt.Errorf("ws aggTrade: normal_qty: %w", err)
		}
	} else {
		normalQty = qty
	}

	out := schema.AggTrade{
		EventTime:    eventTime,
		Symbol:       parseStr("s"),
		AggTradeID:   aggTradeID,
		Price:        price,
		Quantity:     qty,
		NormalQty:    normalQty,
		FirstTradeID: firstTradeID,
		LastTradeID:  lastTradeID,
		TransactTime: transactTime,
		IsBuyerMaker: parseBool("m"),
	}
	payload, err := json.Marshal(out)
	if err != nil {
		return 0, nil, fmt.Errorf("ws aggTrade: marshal: %w", err)
	}
	return transactTime, payload, nil
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
