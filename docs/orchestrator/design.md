# Data Orchestrator Design

## Overview

The orchestrator loads market data from disk (historical) or a live feed (realtime),
merges it globally by timestamp across all symbols, and publishes it to NATS so that
all registered components receive data in a globally consistent order — critical for
cross-symbol correlation backtests.

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
```

---

## File Path Resolution

```
AEGIS_DATA_ROOT=~/media/external_hdd/data   (env var, global config)

klines:     <root>/<SYMBOL>/klines/<timeframe>/<SYMBOL>-<timeframe>-<YYYY-MM-DD>.csv
flat types: <root>/<SYMBOL>/<data_type>/<SYMBOL>-<data_type>-<YYYY-MM-DD>.csv
```

Files for the same topic are sorted chronologically by date in the filename.
Only **one file per topic is loaded into memory at a time** — when a file is
fully published, it is released and the next one is loaded.

---

## Row Priority (same timestamp)

When multiple data types share the same timestamp, they are published in this order:

```
1. trades        (tick-level, most granular)
2. aggTrades     (aggregation of trades)
3. klines        (OHLCV, derived from trades)
4. bookDepth     (order book snapshot)
5. metrics       (position metrics, computed last)
```

---

## Architecture

```
Orchestrator
  │
  ├── GlobalClock                         — owns the current timestamp, drives the barrier
  │     └── barrier                       — WaitGroup-style: all SymbolMergers must
  │                                         confirm ts before clock advances
  │
  ├── SymbolMerger: BTCUSDT               — one goroutine per symbol
  │     ├── DataSource: klines.1m         — one goroutine per data type
  │     ├── DataSource: aggTrades
  │     ├── DataSource: trades
  │     ├── DataSource: bookDepth
  │     └── DataSource: metrics
  │
  ├── SymbolMerger: ETHUSDT
  │     └── ...
  │
  └── NATSPublisher                       — shared, one conn for all symbols
```

### GlobalClock

The GlobalClock collects the **minimum next timestamp** reported by all
SymbolMergers and broadcasts it as the current tick. The flow per tick:

```
1. Each SymbolMerger peeks at its next row timestamp and reports it to the clock.
2. GlobalClock picks the global minimum → that is the next tick T.
3. GlobalClock broadcasts T to all SymbolMergers.
4. Each SymbolMerger publishes all rows where ts == T (in priority order),
   then sends "done" back to the clock.
5. Clock waits for all "done" signals → advances to next tick.
6. Repeat until all DataSources are exhausted.
```

If a SymbolMerger has no rows at tick T, it immediately sends "done" without
publishing anything. No placeholder message is sent to NATS.

### SymbolMerger

Owns all DataSources for one symbol. On each tick T received from the clock:

```
1. Pull all rows with ts == T from each DataSource.
2. Sort them by priority (data_type order).
3. Publish each row to its NATS topic.
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
    // If the next row has a different ts, returns empty slice.
    Drain(timestamp int64) ([]RawRow, error)

    // Topic returns the full NATS topic string for this source.
    Topic() string
}
```

`CSVDataSource` implements this by holding one file fully loaded in memory as
`[]RawRow`, with a cursor. When the cursor reaches the end, it loads the next
file for that topic.

`LiveDataSource` implements the same interface backed by a channel fed by a
WebSocket adapter — identical contract, drop-in replacement.

### RawRow

```go
type RawRow struct {
    Timestamp  int64           // canonical unix ms, normalized from any source format
    DataType   string          // "klines", "aggTrades", "trades", "bookDepth", "metrics"
    Priority   int             // derived from DataType at parse time
    Payload    []byte          // JSON-encoded typed struct
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

`ts` is always unix ms regardless of the source format — `bookDepth` and `metrics`
use datetime strings on disk but are normalized to unix ms at parse time.

---

## Data Schemas

### klines
```
open_time*, open, high, low, close, volume, close_time,
quote_volume, count, taker_buy_volume, taker_buy_quote_volume, ignore
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

### bookDepth
```
timestamp*, percentage, depth, notional
```
`timestamp` format on disk: `2006-01-02 15:04:05` → normalized to unix ms

### metrics
```
create_time*, symbol, sum_open_interest, sum_open_interest_value,
count_toptrader_long_short_ratio, sum_toptrader_long_short_ratio,
count_long_short_ratio, sum_taker_long_short_vol_ratio
```
`create_time` format on disk: `2006-01-02 15:04:05` → normalized to unix ms

---

## Package Layout

```
internals/
  orchestrator/
    orchestrator.go       — Orchestrator struct, Start/Stop, fan-out of SymbolMergers
    clock.go              — GlobalClock: barrier logic, tick broadcast
    merger.go             — SymbolMerger: per-symbol tick handler
    source.go             — DataSource interface + RawRow
    csv_source.go         — CSVDataSource: file loading, cursor, Peek/Drain
    live_source.go        — LiveDataSource: WS channel adapter (future)
    resolver.go           — topic string → []filepath (glob + sort)
    publisher.go          — NATSPublisher: thin wrapper, Publish(topic, []byte)
    envelope.go           — BuildEnvelope(sessionID, topic, ts, data) → []byte
    schema/
      kline.go            — Kline struct + Parse([]string) (RawRow, error)
      aggtrade.go
      trade.go
      bookdepth.go
      metrics.go
```

---

## GlobalClock Barrier — Detailed Flow

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
            │ publish              │ skip               │ publish
            │                     │                    │
            └──────── done ────────┴──────── done ──────┘
                                   │
                            clock advances to T+1
```

The barrier uses a `chan struct{}` per SymbolMerger. The clock sends the tick,
each merger sends back on its done channel. Clock fans out and collects with a
simple loop — no sync.WaitGroup needed since the number of mergers is fixed at
session start.

---

## Error Strategy

| Situation                          | Behavior                                              |
|------------------------------------|-------------------------------------------------------|
| File not found for a topic         | DataSource is skipped, warning logged, clock still advances |
| Malformed CSV row                  | Row skipped, error logged with file + line number     |
| NATS publish error                 | Retry once; on second failure cancel the orchestrator |
| SymbolMerger hangs (no done signal)| GlobalClock has a per-tick timeout (configurable, default 30s) |
| All DataSources exhausted          | Orchestrator calls session.SetToFinished()            |
| Context cancelled                  | All mergers drain current tick and exit cleanly       |