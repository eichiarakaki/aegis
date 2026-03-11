package orchestrator

import (
	"context"
	"sync"
)

const wsRingBufferSize = 8192

// WSDataSource implements DataSource for a live Binance WebSocket stream.
//
// Rows arrive asynchronously from WSManager.dispatch() via Push().
// Peek() and Drain() are called synchronously by the GlobalClock —
// the same contract as CSVDataSource.
//
// Unlike CSVDataSource, Peek() blocks until data arrives rather than
// returning ErrExhausted. ErrExhausted is only returned after Close()
// is called (i.e. the session is stopped).
type WSDataSource struct {
	topic    string
	dataType string
	priority int

	mu     sync.Mutex
	buf    []RawRow
	ready  chan struct{} // receives a token whenever a row is pushed
	closed bool
}

// NewWSDataSource creates a WSDataSource for the given NATS topic.
func NewWSDataSource(topic, dataType string, priority int) *WSDataSource {
	return &WSDataSource{
		topic:    topic,
		dataType: dataType,
		priority: priority,
		buf:      make([]RawRow, 0, wsRingBufferSize),
		ready:    make(chan struct{}, 1),
	}
}

// Topic implements DataSource.
func (s *WSDataSource) Topic() string { return s.topic }

// DataType implements DataSource.
func (s *WSDataSource) DataType() string { return s.dataType }

// Push appends a row to the buffer and signals any blocked Peek.
// Called from the WSManager goroutine.
func (s *WSDataSource) Push(row RawRow) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	// Evict oldest entry when the buffer is full.
	if len(s.buf) >= wsRingBufferSize {
		s.buf = s.buf[1:]
	}
	s.buf = append(s.buf, row)

	// Non-blocking send: if the channel already has a token, a waiter will
	// wake up on the next receive regardless.
	select {
	case s.ready <- struct{}{}:
	default:
	}
}

// Close marks the source as done. Subsequent Peek calls return ErrExhausted
// once the buffer is drained.
func (s *WSDataSource) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	select {
	case s.ready <- struct{}{}:
	default:
	}
}

// Peek implements DataSource.
// Blocks until at least one row is buffered or the source is closed.
func (s *WSDataSource) Peek() (int64, error) {
	for {
		s.mu.Lock()
		if len(s.buf) > 0 {
			ts := s.buf[0].Timestamp
			s.mu.Unlock()
			return ts, nil
		}
		if s.closed {
			s.mu.Unlock()
			return 0, ErrExhausted
		}
		ready := s.ready
		s.mu.Unlock()

		<-ready
	}
}

// PeekCtx is like Peek but unblocks when ctx is cancelled.
func (s *WSDataSource) PeekCtx(ctx context.Context) (int64, error) {
	for {
		s.mu.Lock()
		if len(s.buf) > 0 {
			ts := s.buf[0].Timestamp
			s.mu.Unlock()
			return ts, nil
		}
		if s.closed {
			s.mu.Unlock()
			return 0, ErrExhausted
		}
		ready := s.ready
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ready:
		}
	}
}

// Drain implements DataSource.
// Consumes all buffered rows whose Timestamp == ts.
func (s *WSDataSource) Drain(ts int64) ([]RawRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.buf) == 0 || s.buf[0].Timestamp != ts {
		return nil, nil
	}

	var out []RawRow
	for len(s.buf) > 0 && s.buf[0].Timestamp == ts {
		out = append(out, s.buf[0])
		s.buf = s.buf[1:]
	}
	return out, nil
}
