Infrastructure
==============

This document covers the technology decisions behind Aegis and the end-to-end
workflow for running sessions, from fetching data to interacting with live components.

---

Technology Criteria
-------------------

### Go — daemon and tooling (aegisd, aegisctl, aegis-fetcher)

The daemon needs to manage many concurrent components, each with its own
lifecycle goroutine, I/O loop, and heartbeat ticker. Go's goroutine scheduler
handles this cheaply — thousands of concurrent lightweight threads with no
manual thread pool management. The standard library covers everything the daemon
needs: Unix sockets, HTTP, JSON, process management, and signal handling, without
pulling in external dependencies.

Compilation produces a single static binary per target. Deployment is copying a
file. No runtime, no interpreter, no dependency resolution on the target machine.
Cross-compilation to linux/amd64, linux/arm64, darwin/amd64, and darwin/arm64 is
a one-liner, which is why the release pipeline builds all four.

The garbage collector is not a concern at the daemon level. The hot path —
delivering data frames to components — goes through a Unix socket write, not a
tight compute loop. GC pauses at the microsecond level do not matter here.

### Rust — components (libaegis)

Components are where latency actually matters. A strategy processing aggTrades
at high frequency cannot afford GC pauses, and it owns its memory layout
explicitly because cache behavior matters. Rust gives deterministic memory
management with no runtime overhead, which is the right tradeoff for
compute-intensive strategy logic.

The other reason is correctness. A strategy that mishandles an order state
transition silently is worse than one that fails to compile. Rust's type system
and ownership model make a large class of bugs impossible to ship — especially
useful for financial logic where silent errors are the dangerous ones.

Aegis is used in two contexts: by the developer who maintains the system, and by
third-party teams who use it as a runtime for their own strategies. Both write
components. libaegis abstracts the Aegis protocol (registration, handshake,
lifecycle commands, data stream connection) so neither audience deals with sockets
or JSON framing directly. An internal developer and an external strategy team
both write business logic against the same API — the runtime contract is
libaegis's responsibility, not theirs.

### NATS — internal message bus

The orchestrator fans data out to multiple components simultaneously. A simple
channel-per-component design does not scale when sessions have many components
subscribing to overlapping topic sets. NATS handles the pub/sub routing, topic
filtering, and fan-out natively, and it runs embedded — no separate NATS server
process, no network hop, no configuration.

The alternative (direct channel passing) would require the orchestrator to
maintain an explicit subscriber list per topic and manage channel backpressure
manually. NATS does this already, and the overhead of going through it is
negligible compared to the I/O cost of writing to a Unix socket.

### Unix domain sockets — component transport

The control protocol (lifecycle, commands, heartbeats) and the data stream both
use Unix sockets. The decision is straightforward: components run on the same
machine as the daemon. Unix sockets avoid the TCP stack entirely — no address
resolution, no connection overhead, no Nagle algorithm, no kernel TCP buffer
management. A write to a Unix socket is a memory copy.

They also have a useful operational property: a dead socket file is immediately
visible on the filesystem, and `ss -x` or `lsof` gives a complete picture of
who is connected without any daemon-side tooling.

### Single machine

Aegis is intentionally single-machine. Distributed coordination introduces clock
synchronization problems, network partitions, and consensus overhead that are
simply not worth it for a system where all components are trusted processes
running under the same operator. The GlobalClock barrier in historical mode works
precisely because all data sources and all components share a single process space
and a single wall clock.

If a strategy needs more compute than one machine can provide, the right answer
is to run multiple independent Aegis instances, not to distribute a single session
across machines.

---

Usage Workflow
--------------

### 1. Install and start the daemon

Build from source or download a release binary for your platform:

  $ aegisctl daemon start

The daemon listens on a Unix socket (default: /tmp/aegis.sock) and stays in the
foreground. All session state is kept in memory — there is no database. If the
daemon restarts, sessions are gone and components need to reconnect.

### 2. Fetch historical data

Before running a historical session, download the CSV files for the symbols and
date range you need:

  $ aegis-fetcher \
      --symbol BTCUSDT,ETHUSDT \
      --datatype klines,aggTrades \
      --interval 1m,1h \
      --from 2024-01-01 \
      --to   2024-03-31 \
      --save /data/binance

This downloads daily CSV files from Binance Vision, verifies their SHA-256
checksums, and extracts the zip archives. The resulting directory structure
matches what the orchestrator expects:

  /data/binance/
    BTCUSDT/
      klines/1m/BTCUSDT-1m-2024-01-01.csv
      klines/1h/BTCUSDT-1h-2024-01-01.csv
      aggTrades/BTCUSDT-aggTrades-2024-01-01.csv
    ETHUSDT/
      ...

Set AEGIS_DATA_ROOT to point to this path, or configure it in aegis.yaml:

  data_path: /data/binance

Binance Vision files are typically 2–3 days behind the current date. For data
more recent than that, use a realtime session instead (see section 6).

### 3. Create a session

  $ aegisctl session create --name backtest-q1

This returns a session ID (e.g. sess-a1b2c3). Sessions start in INITIALIZED
state — no components attached, no data flowing yet.

To create a realtime session:

  $ aegisctl session create --name live-btc --mode realtime

The mode is fixed at creation time and cannot be changed afterwards.

### 4. Attach components

  $ aegisctl session attach sess-a1b2c3 \
      --path ./target/release/my-strategy

Each attached binary is registered as a component. The daemon launches it when
the session starts. Components declare their topic subscriptions during the
libaegis handshake — aegisd does not need to know what data they consume at
attach time.

Multiple components can be attached to a single session:

  $ aegisctl session attach sess-a1b2c3 --path ./target/release/risk-manager

For third-party components distributed as binaries, the workflow is the same —
attach the binary and pass configuration via --env. The component only needs to
link against libaegis and implement the standard handshake.

### 5. Start a historical session (backtest)

  $ aegisctl session start sess-a1b2c3 \
      --from 2024-01-01 \
      --to   2024-03-31

The daemon launches all attached binaries. Each component connects back to the
daemon, completes the libaegis handshake (REGISTERED → INITIALIZING → READY →
CONFIGURED → RUNNING), and connects to the data stream socket.

Once all components are RUNNING, the orchestrator starts the GlobalClock. Data
flows in globally ordered ticks until all CSV files for the requested date range
are exhausted. The session transitions automatically to FINISHED.

The component side (libaegis) looks roughly like this:

  let mut component = Component::connect("/tmp/aegis.sock").await?;
  component.declare_topics(&["klines.BTCUSDT.1m", "aggTrades.BTCUSDT"]).await?;

  while let Some(msg) = component.next_message().await? {
      match msg.data_type.as_str() {
          "klines"    => handle_kline(msg),
          "aggTrades" => handle_agg_trade(msg),
          _ => {}
      }
  }

The message schema is identical regardless of whether the session is historical
or realtime, so a component written for backtesting runs unmodified in a live
session.

### 6. Start a realtime session

  $ aegisctl session start sess-live

No --from / --to flags. The orchestrator connects to Binance WebSocket streams
for every topic declared by the attached components. Data starts flowing
immediately and continues until the session is stopped.

To stop it:

  $ aegisctl session stop sess-live

### 7. Inspect state and interact with live sessions

List all sessions:

  $ aegisctl session list

  ID            NAME           MODE        STATE     COMPONENTS
  sess-a1b2c3   backtest-q1    historical  FINISHED  2
  sess-live     live-btc       realtime    RUNNING   1

Inspect a specific session:

  $ aegisctl session inspect sess-live

  Session:    sess-live
  Mode:       realtime
  State:      RUNNING
  Started:    2024-04-01 09:00:03
  Components:
    cmp-e2d4ea2e  my-strategy  RUNNING   uptime=4m32s

Query component health:

  $ aegisctl component health cmp-e2d4ea2e

  Component:  my-strategy
  State:      RUNNING
  Uptime:     4m32s
  Last ping:  312ms ago

Components receive periodic Ping commands from the daemon and respond with Pong.
If a component stops responding, the daemon marks it as unhealthy and logs the
event. The session itself keeps running — a single unresponsive component does
not take down the others.

### 8. Restart a finished session

After a historical session reaches FINISHED, it can be restarted over a
different date range without relaunching the component binaries. The components
receive a REBORN signal that clears their internal state, then the orchestrator
starts a new run:

  $ aegisctl session restart sess-a1b2c3 \
      --from 2024-04-01 \
      --to   2024-06-30

This is significantly faster than creating a new session because the component
processes are already running and connected — only the orchestrator and data
sources are rebuilt. This is the recommended pattern for iterating on a strategy
across multiple date ranges without recompiling or relaunching anything.