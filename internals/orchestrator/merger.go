package orchestrator

import (
	"context"
	"sort"
)

// SymbolMerger owns all DataSources for a single symbol.
// It receives ticks from the GlobalClock, drains all sources at that
// timestamp in priority order, publishes to NATS, then signals done.
type SymbolMerger struct {
	symbol    string
	sessionID string
	sources   []DataSource // all data types for this symbol
	pub       *Publisher

	// Channels wired by GlobalClock.
	tickCh chan tickMsg  // receives tick (ts) or control signals from clock
	peekCh chan peekMsg  // sends peek result back to clock
	doneCh chan struct{} // signals tick completion to clock

	idx int // index in the GlobalClock's merger list
}

// newSymbolMerger creates a SymbolMerger and wires it to the GlobalClock channels.
func newSymbolMerger(
	idx int,
	symbol string,
	sessionID string,
	sources []DataSource,
	pub *Publisher,
	gc *GlobalClock,
) *SymbolMerger {
	return &SymbolMerger{
		idx:       idx,
		symbol:    symbol,
		sessionID: sessionID,
		sources:   sources,
		pub:       pub,
		tickCh:    gc.tickChans[idx],
		peekCh:    gc.peekChans[idx],
		doneCh:    gc.doneChans[idx],
	}
}

// Run starts the merger loop. It blocks until a shutdown signal is received
// or the context is cancelled.
func (m *SymbolMerger) Run(ctx context.Context) {
	exhausted := make([]bool, len(m.sources))

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-m.tickCh:
			switch msg.ts {
			case -2:
				// Peek request: report the minimum next timestamp across all sources.
				m.peekCh <- peekMsg{
					mergerIdx: m.idx,
					ts:        m.minPeek(exhausted),
				}

			case -1:
				// Shutdown signal.
				return

			default:
				// Real tick: drain all sources at ts, publish in priority order.
				m.processTick(ctx, msg.ts, exhausted)
				m.doneCh <- struct{}{}
			}
		}
	}
}

// minPeek returns the minimum next timestamp across all non-exhausted sources.
// Returns 0 if all sources are exhausted.
func (m *SymbolMerger) minPeek(exhausted []bool) int64 {
	var min int64
	for i, src := range m.sources {
		if exhausted[i] {
			continue
		}
		ts, err := src.Peek()
		if err == ErrExhausted {
			exhausted[i] = true
			continue
		}
		if err != nil {
			continue
		}
		if min == 0 || ts < min {
			min = ts
		}
	}
	return min
}

// processTick drains all sources at ts and publishes rows in priority order.
func (m *SymbolMerger) processTick(ctx context.Context, ts int64, exhausted []bool) {
	var rows []RawRow

	for i, src := range m.sources {
		if exhausted[i] {
			continue
		}

		drained, err := src.Drain(ts)
		if err != nil {
			// Non-fatal: log and continue.
			continue
		}
		rows = append(rows, drained...)
	}

	if len(rows) == 0 {
		return
	}

	// Sort by priority (ascending — lower number = higher priority = published first).
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Priority < rows[j].Priority
	})

	for _, row := range rows {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := m.pub.Publish(m.sessionID, row); err != nil {
			// On publish failure cancel is handled at the orchestrator level.
			// Here we just stop publishing this tick.
			return
		}
	}
}
