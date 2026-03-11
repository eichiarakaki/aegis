# Data Orchestrator Design

## Overview

The orchestrator loads market data from disk (historical) or a live Binance
WebSocket feed (realtime), and publishes it to connected components via a
Unix socket (`DataStreamServer`).

In **historical mode** data is merged globally by timestamp across all symbols
using a `GlobalClock` barrier — critical for cross-symbol correlation backtests.

In **realtime mode** there is no clock. Every message is published the instant
it arrives from Binance. Components receive data as fast as the exchange sends
it, with no ordering guarantee between data types.

The session's `Mode` field (`"historical"` | `"realtime"`) determines which
path the orchestrator takes at startup.

---

## Topic Structure

```
aegis.<session_id>.<data_type>.<symbol>.<timeframe>
aegis.<session_id>.<data_type>.<symbol>              ← flat types (no timeframe)
```

Examples:
```
aegis.sess-a1b2c3.klines.BTCUSDT.1m
aegis.sess-a1b2c3.aggTrades.BTCUSDT
aegis.sess-a1b2c3.trades.BTCUSDT
aegis.sess-a1b2c3.bookDepth.BTCUSDT
aegis.sess-a1b2c3.metrics.BTCUSDT
aegis.sess-a1b2c3.orderBook.BTCUSDT     ← realtime only
```

---

## File Path Resolution (historical only)

```
AEGIS_DATA_ROOT=~/media/external_hdd/data   (env var or global config)

klines:     <root>/<SYMBOL>/klines/<timeframe>/<SYMBOL>-<timeframe>-<YYYY-MM-DD>.csv
flat types: <root>/<SYMBOL>/<data_type>/<SYMBOL>-<data_type>-<YYYY-MM-DD>.csv
```

Files for the same topic are sorted chronologically by date in the filename.
Only **one file per topic is loaded into memory at a time** — when a file is
fully published it is released and the next one is loaded.

---

## Row Priority (historical only)

When multiple data types share the same timestamp, they are published in this order:

```
1. trades        (tick-level, most granular)
2. aggTrades     (aggregation of trades)
3. klines        (OHLCV, derived from trades)
4. bookDepth     (order book snapshot)
5. metrics       (position metrics, computed last)
```

Priority has no meaning in realtime mode — rows are published in arrival order.

---

## Architecture

### Historical mode

```
Orchestrator
  │
  ├── GlobalClock                         — owns the current timestamp, drives the barrier
  │     └── barrier                       — all SymbolMergers must confirm ts before clock advances
  │
  ├── SymbolMerger: BTCUSDT               — one goroutine per symbol
  │     ├── CSVDataSource: klines.1m
  │     ├── CSVDataSource: aggTrades
  │     ├── CSVDataSource: trades
  │     ├── CSVDataSource: bookDepth
  │     └── CSVDataSource: metrics
  │
  ├── SymbolMerger: ETHUSDT
  │     └── ...
  │
  ├── Publisher                           — shared, routes rows to DataStreamServer
  └── DataStreamServer                    — Unix socket, delivers to components with backpressure
```

### Realtime mode

```
Orchestrator
  │
  ├── WSManager                           — single combined-stream WebSocket connection
  │     ├── stream: btcusdt@aggTrade      ─┐
  │     ├── stream: btcusdt@kline_1m       ├── parsed and published immediately on arrival
  │     ├── stream: btcusdt@depth20@100ms ─┘
  │     └── ...
  │
  ├── Publisher                           — shared, routes rows to DataStreamServer
  └── DataStreamServer                    — same Unix socket as historical, no backpressure
```

No `GlobalClock`, no `SymbolMerger`, no `CSVDataSource` in realtime mode.

---

## Historical — Component Details

### GlobalClock

Collects the **minimum next timestamp** reported by all SymbolMergers and
broadcasts it as the current tick. Flow per tick:

```
1. Each SymbolMerger peeks at its next row timestamp and reports it to the clock.
2. GlobalClock picks the global minimum → that is the next tick T.
3. GlobalClock broadcasts T to all SymbolMergers.
4. Each SymbolMerger publishes all rows where ts == T (in priority order),
   then sends "done" back to the clock.
5. Clock waits for all "done" signals → advances to next tick.
6. Repeat until all DataSources are exhausted.
```

If a SymbolMerger has no rows at tick T it immediately sends "done" without
publishing anything. No placeholder is sent to NATS.

```
                    ┌─────────────────────────────────────┐
                    │           GlobalClock                │
                    │                                      │
                    │  1. collect peeks from all mergers   │
                    │  2. minTS = min(all peeks)           │
                    │  3. broadcast minTS                  │
                    │  4. wait for N "done" signals        │
                    │  5. goto 1                           │
                    └──────────────┬──────────────────────┘
                                   │ broadcast tick T
               ┌───────────────────┼───────────────────┐
               ▼                   ▼                   ▼
        SymbolMerger          SymbolMerger         SymbolMerger
         BTCUSDT               ETHUSDT              SOLUSDT
          has rows              no rows              has rows
          at T                  at T                 at T
            │                     │                    │
            │ publish             │ skip               │ publish
            │                     │                    │
            └──────── done ───────┴──────── done ──────┘
                                  │
                            clock advances to T+1
```

The barrier uses a `chan struct{}` per SymbolMerger. The clock sends the tick,
each merger sends back on its done channel. No `sync.WaitGroup` needed since
the number of mergers is fixed at session start.

### SymbolMerger

Owns all DataSources for one symbol. On each tick T:

```
1. Pull all rows with ts == T from each DataSource.
2. Sort by priority (data_type order).
3. Publish each row via Publisher.
4. Signal "done" to GlobalClock.
```

Between ticks the SymbolMerger is idle — it does not pre-fetch or buffer ahead.

### DataSource (interface)

```go
type DataSource interface {
    // Peek returns the timestamp of the next row without consuming it.
    // Returns (0, io.EOF) when exhausted.
    Peek() (int64, error)

    // Drain consumes and returns all rows with ts == timestamp.
    // Returns empty slice if next row has a different ts.
    Drain(timestamp int64) ([]RawRow, error)

    // Topic returns the full NATS topic string for this source.
    Topic() string

    // DataType returns the data type string ("klines", "trades", etc.)
    DataType() string
}
```

`CSVDataSource` implements this by holding one file fully in memory as
`[]RawRow` with a cursor. When the cursor reaches the end it loads the next
file for that topic.

---

## Realtime — Component Details

### WSManager

Opens a **single combined-stream WebSocket connection** to Binance for all
topics in the session:

```
wss://fstream.binance.com/stream?streams=btcusdt@aggTrade/btcusdt@kline_1m/...
```

On each incoming message:

```
1. Unwrap the combined-stream envelope: { "stream": "...", "data": { ... } }
2. Look up the matching wsSubscription by stream name.
3. Parse the raw JSON payload using the data-type-specific WSParseFunc.
4. Build a RawRow and call Publisher.Publish() immediately.
```

No buffering, no ordering, no clock. If Binance closes the connection,
WSManager reconnects with exponential backoff (3s → 60s max).

### Why no clock in realtime

Two reasons make a GlobalClock unsuitable for realtime:

**Head-of-line blocking.** `aggTrades` arrive every few milliseconds while
`klines` arrive roughly once per second. A clock barrier would withhold all
`aggTrades` until the next `kline` tick — destroying the low-latency benefit
of a live feed.

**No true backpressure.** Binance cannot be paused. Any buffer between the
WebSocket and the clock would silently drop rows when a slow component fills
it. Genuine backpressure is not achievable against an external WebSocket.

---

## Publisher

Routes a `RawRow` to the `DataStreamServer`, which delivers it to all
subscribed components via the Unix socket.

```go
// Historical: Deliver() blocks until every subscriber has received the message
//             → natural backpressure that keeps the clock in sync with components.
// Realtime:   Deliver() is called from the WSManager goroutine without a clock
//             barrier → a slow component does not stall the WebSocket reader,
//             but may miss messages if it cannot keep up.
func (p *Publisher) Publish(sessionID string, row RawRow) error
```

In both modes the component connects to the same Unix socket path and uses the
same handshake. The mode is transparent to the component.

---

## RawRow

```go
type RawRow struct {
    Timestamp int64  // canonical unix ms, normalized from any source format
    DataType  string // "klines", "aggTrades", "trades", "bookDepth", "metrics", "orderBook"
    Priority  int    // meaningful in historical only (used for same-ts ordering)
    Topic     string // full NATS topic: aegis.<sid>.<type>.<sym>[.<tf>]
    Payload   []byte // JSON-encoded typed struct (same schema in both modes)
}
```

---

## NATS Message Envelope

```json
{
  "session_id": "sess-a1b2c3",
  "topic":      "aegis.sess-a1b2c3.klines.BTCUSDT.1m",
  "ts":         1767139200000,
  "data": {
    "open_time": 1767139200000,
    "open":  88455.20,
    "high":  88455.30,
    "low":   88381.80,
    "close": 88403.10,
    ...
  }
}
```

`ts` is always unix ms regardless of source format — `bookDepth` and `metrics`
use datetime strings on disk but are normalized to unix ms at parse time.
WebSocket payloads use the same schema structs as CSV parsers — the envelope
is identical in both modes.

---

## Data Schemas

### klines
```
open_time*, open, high, low, close, volume, close_time,
quote_volume, count, taker_buy_volume, taker_buy_quote_volume
```
`*` = timestamp field (unix ms)

### aggTrades
```
agg_trade_id, price, quantity, first_trade_id, last_trade_id, transact_time*, is_buyer_maker
```

### trades
```
id, price, qty, quote_qty, time*, is_buyer_maker
```
> **Realtime note:** `quote_qty` is not provided by the WebSocket trade stream.
> It is approximated as `price × qty`.

### bookDepth _(historical only)_
```
timestamp*, percentage, depth, notional
```
`timestamp` on disk: `2006-01-02 15:04:05` → normalized to unix ms

### metrics _(historical only)_
```
create_time*, symbol, sum_open_interest, sum_open_interest_value,
count_toptrader_long_short_ratio, sum_toptrader_long_short_ratio,
count_long_short_ratio, sum_taker_long_short_vol_ratio
```
`create_time` on disk: `2006-01-02 15:04:05` → normalized to unix ms

### orderBook _(realtime only)_
```json
{
  "last_update_id": 160,
  "event_time": 1767139200000,
  "bids": [{ "price": 88400.0, "quantity": 1.5 }, ...],
  "asks": [{ "price": 88401.0, "quantity": 0.8 }, ...]
}
```
20 levels, updated every 100ms. No CSV equivalent.

---

## Package Layout

```
internals/
  orchestrator/
    orchestrator.go     — Orchestrator struct, Start/Stop, historical/realtime dispatch
    clock.go            — GlobalClock: barrier logic, tick broadcast (historical only)
    merger.go           — SymbolMerger: per-symbol tick handler (historical only)
    source.go           — DataSource interface, RawRow, parser registry
    csv_source.go       — CSVDataSource: file loading, cursor, Peek/Drain
    ws_source.go        — WSDataSource: ring-buffered source for future clocked WS use
    ws_manager.go       — WSManager: combined-stream WebSocket, reconnect, dispatch
    ws_parser.go        — WSParseFunc per data type, json.Number for OHLCV fields
    resolver.go         — topic string → []filepath (glob + sort + date range filter)
    publisher.go        — Publisher: routes RawRow to DataStreamServer or NATS
    data_stream.go      — DataStreamServer: Unix socket server, per-component delivery
    schema/
      kline.go          — Kline struct + ParseKline([]string)
      aggtrade.go       — AggTrade struct + ParseAggTrade([]string)
      trade.go          — Trade struct + ParseTrade([]string)
      bookdepth.go      — BookDepth struct + ParseBookDepth([]string)
      metrics.go        — Metrics struct + ParseMetrics([]string)
      orderbook.go      — OrderBook + PriceLevel structs + ParseOrderBook([]byte)
```

---

## Error Strategy

| Situation | Behavior |
|---|---|
| File not found for a topic | DataSource skipped, warning logged, clock still advances |
| Malformed CSV row | Row skipped, error logged with file + line number |
| Publish error (DataStreamServer) | Logged as warning; realtime continues, historical retries once then cancels |
| SymbolMerger hangs (no done signal) | GlobalClock has a per-tick timeout (default 30s) |
| All DataSources exhausted | Orchestrator calls `OnFinished` → session transitions to FINISHED |
| WebSocket disconnected | WSManager reconnects with exponential backoff (3s → 60s) |
| `bookDepth` requested in realtime | Error returned at session start — use `orderBook` instead |
| `orderBook` requested in historical | Error returned at session start — no CSV equivalent |
| Context cancelled | Historical: mergers drain current tick and exit. Realtime: WSManager stops reading. |