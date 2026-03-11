package orchestrator

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/nats-io/nats.go"
)

// Config holds everything the Orchestrator needs to start.
// Mode is read directly from core.Session.Mode ("realtime" | "historical").
type Config struct {
	SessionID string
	Topics    []string
	DataRoot  string // used only when Mode == "historical"
	NC        *nats.Conn
	DS        *DataStreamServer // Unix socket delivery to components

	// Mode mirrors core.Session.Mode: "realtime" or "historical".
	// Any value other than "realtime" is treated as historical.
	Mode string

	// Historical time range (unix ms, inclusive). Ignored when Mode == "realtime".
	FromTS int64
	ToTS   int64
}

func (c Config) isRealtime() bool { return c.Mode == "realtime" }

// Orchestrator fans out one SymbolMerger per unique symbol (historical), or
// starts a WSManager that publishes all rows immediately (realtime).
//
// In realtime mode there is no GlobalClock. Every message received from
// Binance is published the instant it arrives via the DataStreamServer
// (same Unix socket path as historical — components connect identically).
// The only difference from historical is the absence of backpressure:
// Deliver() is called from the WSManager goroutine without waiting for a
// clock barrier, so a slow component will not stall the WebSocket reader.
type Orchestrator struct {
	cfg    Config
	cancel context.CancelFunc
	wg     sync.WaitGroup

	wsManager *WSManager // non-nil in realtime mode

	// OnFinished is called when all CSV sources are exhausted (historical only).
	OnFinished func()
	// OnError is called on a fatal clock or merger error (historical only).
	OnError func(err error)
}

// New creates an Orchestrator. OnFinished and OnError can be set between
// New and Start.
func New(cfg Config) (*Orchestrator, error) {
	if !cfg.isRealtime() && cfg.DataRoot == "" {
		dataRoot, err := dataRootFromEnv()
		if err != nil {
			return nil, err
		}
		cfg.DataRoot = dataRoot
	}
	return &Orchestrator{cfg: cfg}, nil
}

// Start builds sources and launches goroutines. Returns immediately.
func (o *Orchestrator) Start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	o.cancel = cancel

	if o.cfg.isRealtime() {
		return o.startRealtime(runCtx)
	}
	return o.startHistorical(runCtx)
}

// Stop cancels all goroutines, stops the WSManager, and waits for clean exit.
func (o *Orchestrator) Stop() {
	if o.cancel != nil {
		o.cancel()
	}
	if o.wsManager != nil {
		o.wsManager.Stop()
	}
	o.wg.Wait()
}

// ── realtime ─────────────────────────────────────────────────────────────────

func (o *Orchestrator) startRealtime(ctx context.Context) error {
	// Use DS so rows reach components via the Unix socket, exactly like
	// historical mode. The difference is that Deliver() is called directly
	// from the WSManager goroutine — no clock barrier, no backpressure.
	pub := NewPublisher(o.cfg.NC, o.cfg.DS)

	subs, err := o.buildRealtimeSubs(pub)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		return fmt.Errorf("orchestrator realtime: no streamable topics for session %s", o.cfg.SessionID)
	}

	mgr := NewWSManager(o.cfg.SessionID, subs)
	mgr.Start(ctx)
	o.wsManager = mgr

	return nil
}

func (o *Orchestrator) buildRealtimeSubs(pub *Publisher) ([]wsSubscription, error) {
	var subs []wsSubscription

	for _, rawTopic := range o.cfg.Topics {
		tp, err := ParseTopic(rawTopic)
		if err != nil {
			return nil, fmt.Errorf("topic %q: %w", rawTopic, err)
		}

		if tp.DataType == "bookDepth" {
			return nil, fmt.Errorf(
				"topic %q: \"bookDepth\" is not available in realtime mode — "+
					"use \"orderBook\" for live order book data", rawTopic,
			)
		}

		_, priority, err := DataTypeInfo(tp.DataType)
		if err != nil {
			return nil, fmt.Errorf("topic %q: %w", rawTopic, err)
		}

		parseFn, err := WSParserFor(tp.DataType)
		if err != nil {
			// Data type has no WS stream (e.g. "metrics") — skip gracefully.
			continue
		}

		stream, err := streamName(tp.DataType, tp.Symbol, tp.Timeframe)
		if err != nil {
			return nil, fmt.Errorf("topic %q: %w", rawTopic, err)
		}

		subs = append(subs, wsSubscription{
			streamName: stream,
			dataType:   tp.DataType,
			priority:   priority,
			natsTopic:  NATSTopic(o.cfg.SessionID, tp),
			parseFn:    parseFn,
			pub:        pub,
			sessionID:  o.cfg.SessionID,
		})
	}

	return subs, nil
}

// ── historical ───────────────────────────────────────────────────────────────

func (o *Orchestrator) startHistorical(ctx context.Context) error {
	pub := NewPublisher(o.cfg.NC, o.cfg.DS)
	resolver := NewFileResolverWithRange(o.cfg.DataRoot, o.cfg.FromTS, o.cfg.ToTS)

	symbolSources, err := o.buildHistoricalSources(resolver)
	if err != nil {
		return fmt.Errorf("orchestrator: build sources: %w", err)
	}
	if len(symbolSources) == 0 {
		return fmt.Errorf("orchestrator: no valid data sources for session %s", o.cfg.SessionID)
	}

	symbols := make([]string, 0, len(symbolSources))
	for sym := range symbolSources {
		symbols = append(symbols, sym)
	}

	gc := newGlobalClock(len(symbols))
	for i, sym := range symbols {
		m := newSymbolMerger(i, sym, o.cfg.SessionID, symbolSources[sym], pub, gc)
		o.wg.Add(1)
		go func(m *SymbolMerger) {
			defer o.wg.Done()
			m.Run(ctx)
		}(m)
	}

	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		err := gc.Run(ctx)
		o.cancel()
		if err != nil && err != context.Canceled {
			if o.OnError != nil {
				o.OnError(err)
			}
			return
		}
		if o.OnFinished != nil {
			o.OnFinished()
		}
	}()

	return nil
}

func (o *Orchestrator) buildHistoricalSources(resolver *FileResolver) (map[string][]DataSource, error) {
	symbolSources := make(map[string][]DataSource)

	for _, rawTopic := range o.cfg.Topics {
		tp, err := ParseTopic(rawTopic)
		if err != nil {
			return nil, fmt.Errorf("topic %q: %w", rawTopic, err)
		}

		parseFn, priority, err := DataTypeInfo(tp.DataType)
		if err != nil {
			return nil, fmt.Errorf("topic %q: %w", rawTopic, err)
		}
		if parseFn == nil {
			return nil, fmt.Errorf(
				"topic %q: data type %q has no CSV representation and cannot be used in historical mode",
				rawTopic, tp.DataType,
			)
		}

		files, err := resolver.Resolve(tp)
		if err != nil {
			continue // non-fatal: skip topics with no matching files
		}

		natsTopic := NATSTopic(o.cfg.SessionID, tp)
		src := NewCSVDataSourceWithRange(natsTopic, tp.DataType, priority, parseFn, files, o.cfg.FromTS, o.cfg.ToTS)
		symbolSources[tp.Symbol] = append(symbolSources[tp.Symbol], src)
	}

	return symbolSources, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func dataRootFromEnv() (string, error) {
	if v := os.Getenv("AEGIS_DATA_ROOT"); v != "" {
		return v, nil
	}
	cfg, err := config.LoadAegis()
	if err != nil {
		return "", err
	}
	return cfg.DataPath, nil
}
