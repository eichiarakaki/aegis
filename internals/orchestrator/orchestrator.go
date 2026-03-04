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
type Config struct {
	SessionID string
	Topics    []string // component topic strings, e.g. ["klines.BTCUSDT.1m", "trades.BTCUSDT"]
	DataRoot  string   // AEGIS_DATA_ROOT
	NC        *nats.Conn
}

// Orchestrator fans out one SymbolMerger per unique symbol found in Topics,
// wires them to a GlobalClock, and drives the tick loop.
type Orchestrator struct {
	cfg    Config
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// OnFinished is called when all data sources are exhausted.
	// Wire this to session.SetToFinished() at the call site.
	OnFinished func()

	// OnError is called if the clock or a merger returns a fatal error.
	OnError func(err error)
}

// New creates an Orchestrator. OnFinished and OnError can be set after New
// and before Start.
func New(cfg Config) (*Orchestrator, error) {
	if cfg.DataRoot == "" {

		DataRoot, err := dataRootFromEnv()
		if err != nil {
			return nil, err
		}
		cfg.DataRoot = DataRoot
	}
	return &Orchestrator{cfg: cfg}, nil
}

// Start builds the source graph, wires the GlobalClock, and launches all
// goroutines. It returns immediately after the goroutines are running.
func (o *Orchestrator) Start(ctx context.Context) error {
	resolver := NewFileResolver(o.cfg.DataRoot)
	pub := NewPublisher(o.cfg.NC)

	// Group topics by symbol.
	symbolSources, err := o.buildSources(resolver)
	if err != nil {
		return fmt.Errorf("orchestrator: build sources: %w", err)
	}

	if len(symbolSources) == 0 {
		return fmt.Errorf("orchestrator: no valid data sources found for session %s", o.cfg.SessionID)
	}

	// Collect symbols in a deterministic order.
	symbols := make([]string, 0, len(symbolSources))
	for sym := range symbolSources {
		symbols = append(symbols, sym)
	}

	gc := newGlobalClock(len(symbols))

	mergers := make([]*SymbolMerger, len(symbols))
	for i, sym := range symbols {
		mergers[i] = newSymbolMerger(i, sym, o.cfg.SessionID, symbolSources[sym], pub, gc)
	}

	runCtx, cancel := context.WithCancel(ctx)
	o.cancel = cancel

	// Launch SymbolMergers.
	for _, m := range mergers {
		m := m
		o.wg.Add(1)
		go func() {
			defer o.wg.Done()
			m.Run(runCtx)
		}()
	}

	// Launch GlobalClock — drives everything.
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		err := gc.Run(runCtx)
		cancel() // stop all mergers regardless of why the clock stopped

		if err != nil && err != context.Canceled {
			if o.OnError != nil {
				o.OnError(err)
			}
			return
		}

		// Normal exit: all data exhausted.
		if o.OnFinished != nil {
			o.OnFinished()
		}
	}()

	return nil
}

// Stop cancels the orchestrator and waits for all goroutines to exit.
func (o *Orchestrator) Stop() {
	if o.cancel != nil {
		o.cancel()
	}
	o.wg.Wait()
}

// buildSources parses all topics, resolves files, and groups DataSources by symbol.
func (o *Orchestrator) buildSources(resolver *FileResolver) (map[string][]DataSource, error) {
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

		files, err := resolver.Resolve(tp)
		if err != nil {
			// Non-fatal: warn and skip this topic.
			// In production, wire this to the session logger.
			_ = fmt.Sprintf("orchestrator: topic %q: %v (skipped)", rawTopic, err)
			continue
		}

		natsTopic := NATSTopic(o.cfg.SessionID, tp)
		src := NewCSVDataSource(natsTopic, tp.DataType, priority, parseFn, files)

		symbolSources[tp.Symbol] = append(symbolSources[tp.Symbol], src)
	}

	return symbolSources, nil
}

// dataRootFromEnv reads AEGIS_DATA_ROOT from the environment,
// falling back to ~/media/external_hdd/data.
func dataRootFromEnv() (string, error) {
	if v := os.Getenv("AEGIS_DATA_ROOT"); v != "" {
		return v, nil
	}
	// Load from aegis.yaml
	cfg, err := config.LoadAegis()
	// logger.Debug("DATA:", cfg.DataPath)

	if err != nil {
		return "", err
	}

	return cfg.DataPath, nil
}
