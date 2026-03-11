# Data Type Compatibility

This document describes which data types are available in each session mode,
what each type contains, and the known limitations per mode.

---

## Session Modes

| Mode         | Data source                | Delivery                                      |
|--------------|----------------------------|-----------------------------------------------|
| `historical` | CSV files (Binance Vision) | GlobalClock → SymbolMerger → backpressure via Unix socket |
| `realtime`   | Binance WebSocket streams  | Published immediately as messages arrive — no clock, no ordering guarantee |

### Why realtime has no clock

In historical mode a `GlobalClock` collects the minimum timestamp across all
sources before emitting a tick. This works because the data already exists on
disk — global ordering is possible and meaningful.

In realtime this breaks down for two reasons:

1. **Head-of-line blocking.** `klines` updates arrive roughly once per second
   while `aggTrades` arrive every few milliseconds. A clock barrier would
   withhold all `aggTrades` until the next `kline` tick arrives, destroying
   the low-latency benefit of a live feed.

2. **No true backpressure.** Binance cannot be paused. Any buffer between the
   WebSocket and the clock would silently drop rows when a slow component
   causes the buffer to fill. Real backpressure is not achievable against an
   external WebSocket.

In realtime mode every message is published to NATS the instant it is parsed.
Components that need to correlate data types by timestamp must do so
themselves.

---

## Data Types

### `klines`

Candlestick (OHLCV) data at a fixed interval.

| | Historical | Realtime |
|---|---|---|
| **Available** | ✅ | ✅ |
| **Source** | CSV files per day per interval | `<symbol>@kline_<interval>` WebSocket stream |
| **Clock** | GlobalClock | None — clockless |
| **Timestamp field** | `open_time` | `open_time` |
| **Schema** | `Kline` | `Kline` (identical) |
| **Limitations** | Files not available for the last ~2 days on Binance Vision. | Fires on every tick during the candle, not only on close. `close_time` is the projected end of the current candle, not a settled value. |

**Topic format:** `klines.<SYMBOL>.<interval>` — e.g. `klines.BTCUSDT.1m`

---

### `aggTrades`

Aggregated trades — multiple fills at the same price and time merged into one record.

| | Historical | Realtime |
|---|---|---|
| **Available** | ✅ | ✅ |
| **Source** | CSV files per day | `<symbol>@aggTrade` WebSocket stream |
| **Clock** | GlobalClock | None — clockless |
| **Timestamp field** | `transact_time` | `transact_time` |
| **Schema** | `AggTrade` | `AggTrade` (identical) |
| **Limitations** | — | — |

**Topic format:** `aggTrades.<SYMBOL>` — e.g. `aggTrades.BTCUSDT`

---

### `trades`

Individual raw trades (one record per fill).

| | Historical | Realtime |
|---|---|---|
| **Available** | ✅ | ✅ |
| **Source** | CSV files per day | `<symbol>@trade` WebSocket stream |
| **Clock** | GlobalClock | None — clockless |
| **Timestamp field** | `time` | `time` |
| **Schema** | `Trade` | `Trade` (identical) |
| **Limitations** | — | `quote_qty` is not provided by the WebSocket trade stream. It is approximated as `price × qty`, which may differ slightly from the exchange-reported value. |

**Topic format:** `trades.<SYMBOL>` — e.g. `trades.BTCUSDT`

---

### `bookDepth`

Aggregated order book depth snapshots (percentage, depth, notional).
**Historical mode only.**

| | Historical | Realtime |
|---|---|---|
| **Available** | ✅ | ❌ |
| **Source** | CSV files per day | — |
| **Clock** | GlobalClock | — |
| **Timestamp field** | `timestamp` | — |
| **Schema** | `BookDepth` | — |
| **Limitations** | — | Binance does not provide a WebSocket stream with the same aggregated `percentage / depth / notional` semantics. Use `orderBook` for live order book data. Requesting `bookDepth` in realtime mode returns an error at session start. |

**Topic format:** `bookDepth.<SYMBOL>` — e.g. `bookDepth.BTCUSDT`

---

### `orderBook`

Partial order book snapshots (top 20 bid/ask levels).
**Realtime mode only.**

| | Historical | Realtime |
|---|---|---|
| **Available** | ❌ | ✅ |
| **Source** | — | `<symbol>@depth20@100ms` WebSocket stream |
| **Clock** | — | None — clockless |
| **Timestamp field** | — | `event_time` (unix ms); falls back to `last_update_id` when the stream omits it |
| **Schema** | — | `OrderBook` |
| **Limitations** | No CSV equivalent on Binance Vision. | Snapshot-only — 20 levels, updated every 100 ms. Does not support full depth reconstruction. `event_time` may be `0` on endpoints that omit it; in that case `last_update_id` is used as a monotonic proxy and should not be treated as a wall-clock timestamp. |

**Topic format:** `orderBook.<SYMBOL>` — e.g. `orderBook.BTCUSDT`

---

### `metrics`

Open interest, long/short ratios, and taker volume ratio snapshots.
**Historical mode only.**

| | Historical | Realtime |
|---|---|---|
| **Available** | ✅ | ❌ |
| **Source** | CSV files per day | — |
| **Clock** | GlobalClock | — |
| **Timestamp field** | `create_time` | — |
| **Schema** | `Metrics` | — |
| **Limitations** | — | Binance does not provide a public WebSocket stream for open interest or long/short ratio data. These metrics are only available via Binance Vision historical files or REST polling. |

**Topic format:** `metrics.<SYMBOL>` — e.g. `metrics.BTCUSDT`

---

## Summary table

| Data type   | Historical | Realtime | Clock         |
|-------------|:----------:|:--------:|:-------------:|
| `klines`    | ✅ | ✅ | Historical only |
| `aggTrades` | ✅ | ✅ | Historical only |
| `trades`    | ✅ | ✅ | Historical only |
| `bookDepth` | ✅ | ❌ | Historical only |
| `orderBook` | ❌ | ✅ | Never           |
| `metrics`   | ✅ | ❌ | Historical only |

---

## Requesting `bookDepth` in realtime

Aegis returns an error at session start if `bookDepth` is requested in
realtime mode:

```
topic "bookDepth.BTCUSDT": "bookDepth" is not available in realtime mode —
use "orderBook" for live order book data
```

Note that the schemas differ: `BookDepth` stores aggregated
`percentage / depth / notional` values, while `OrderBook` stores raw
bid/ask price levels.

---

## Requesting `orderBook` in historical mode

Aegis returns an error at session start if `orderBook` is requested in
historical mode:

```
topic "orderBook.BTCUSDT": data type "orderBook" has no CSV representation
and cannot be used in historical mode
```