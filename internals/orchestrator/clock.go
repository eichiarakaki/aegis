package orchestrator

import (
	"context"
	"fmt"
	"math"
	"time"
)

const defaultTickTimeout = 30 * time.Second

// tickMsg is sent from the GlobalClock to each SymbolMerger.
type tickMsg struct {
	ts int64 // current tick timestamp (unix ms), or -1 to signal shutdown
}

// peekMsg is sent from each SymbolMerger to the GlobalClock to report
// the next available timestamp. ts == 0 means the merger is exhausted.
type peekMsg struct {
	mergerIdx int
	ts        int64 // 0 = exhausted
}

// GlobalClock coordinates the tick barrier across all SymbolMergers.
// It collects peeks, computes the global minimum timestamp, broadcasts
// the tick, and waits for all mergers to signal done before advancing.
type GlobalClock struct {
	tickTimeout time.Duration

	// One channel pair per SymbolMerger.
	tickChans []chan tickMsg  // clock → merger: "publish this ts"
	peekChans []chan peekMsg  // merger → clock: "my next ts is X"
	doneChans []chan struct{} // merger → clock: "I finished this tick"
}

// newGlobalClock creates a GlobalClock wired to n SymbolMergers.
func newGlobalClock(n int) *GlobalClock {
	gc := &GlobalClock{
		tickTimeout: defaultTickTimeout,
		tickChans:   make([]chan tickMsg, n),
		peekChans:   make([]chan peekMsg, n),
		doneChans:   make([]chan struct{}, n),
	}
	for i := 0; i < n; i++ {
		gc.tickChans[i] = make(chan tickMsg, 1)
		gc.peekChans[i] = make(chan peekMsg, 1)
		gc.doneChans[i] = make(chan struct{}, 1)
	}
	return gc
}

// Run starts the clock loop. It blocks until all mergers are exhausted
// or the context is cancelled.
func (gc *GlobalClock) Run(ctx context.Context) error {
	n := len(gc.tickChans)
	exhausted := make([]bool, n)

	for {
		// --- Phase 1: collect peeks from all active mergers ---
		minTS := int64(math.MaxInt64)
		activeCount := 0

		for i := 0; i < n; i++ {
			if exhausted[i] {
				continue
			}

			// Signal each merger to report its peek.
			// Mergers are already blocked waiting on tickChans, so we
			// send a peek-request by sending ts=-2 (sentinel).
			gc.tickChans[i] <- tickMsg{ts: -2}
		}

		for i := 0; i < n; i++ {
			if exhausted[i] {
				continue
			}

			select {
			case <-ctx.Done():
				gc.shutdown()
				return ctx.Err()
			case pm := <-gc.peekChans[i]:
				if pm.ts == 0 {
					exhausted[i] = true
					continue
				}
				activeCount++
				if pm.ts < minTS {
					minTS = pm.ts
				}
			case <-time.After(gc.tickTimeout):
				return fmt.Errorf("clock: timeout waiting for peek from merger %d", i)
			}
		}

		if activeCount == 0 {
			// All mergers exhausted — signal shutdown and return.
			gc.shutdown()
			return nil
		}

		// --- Phase 2: broadcast the tick to all active mergers ---
		for i := 0; i < n; i++ {
			if exhausted[i] {
				continue
			}
			gc.tickChans[i] <- tickMsg{ts: minTS}
		}

		// --- Phase 3: wait for all active mergers to finish the tick ---
		for i := 0; i < n; i++ {
			if exhausted[i] {
				continue
			}
			select {
			case <-ctx.Done():
				gc.shutdown()
				return ctx.Err()
			case <-gc.doneChans[i]:
			case <-time.After(gc.tickTimeout):
				return fmt.Errorf("clock: timeout waiting for done from merger %d at ts=%d", i, minTS)
			}
		}
	}
}

// shutdown sends a shutdown signal (-1) to all mergers.
func (gc *GlobalClock) shutdown() {
	for _, ch := range gc.tickChans {
		ch <- tickMsg{ts: -1}
	}
}
