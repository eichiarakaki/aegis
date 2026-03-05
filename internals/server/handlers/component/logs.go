package component

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/nats-io/nats.go"
)

// LogEntry is the unit streamed from daemon → CLI over the Unix socket.
type LogEntry struct {
	ComponentID   string    `json:"component_id"`
	ComponentName string    `json:"component_name"`
	Timestamp     time.Time `json:"timestamp"`
	Level         string    `json:"level"`
	Message       string    `json:"message"`
}

// LogBuffer is a fixed-size ring buffer that stores the last N log entries
// for a single component. It is safe for concurrent use.
type LogBuffer struct {
	entries []LogEntry
	size    int
	head    int
	count   int
	mu      sync.RWMutex
}

func NewLogBuffer(size int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

func (b *LogBuffer) Push(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[b.head] = entry
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	}
}

func (b *LogBuffer) Snapshot() []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.count == 0 {
		return nil
	}
	out := make([]LogEntry, b.count)
	start := (b.head - b.count + b.size) % b.size
	for i := 0; i < b.count; i++ {
		out[i] = b.entries[(start+i)%b.size]
	}
	return out
}

// LogStore holds one LogBuffer per component ID.
type LogStore struct {
	buffers    map[string]*LogBuffer
	bufferSize int
	mu         sync.RWMutex
}

func NewLogStore(bufferSize int) *LogStore {
	return &LogStore{
		buffers:    make(map[string]*LogBuffer),
		bufferSize: bufferSize,
	}
}

func (s *LogStore) Push(componentID string, entry LogEntry) {
	s.mu.Lock()
	buf, ok := s.buffers[componentID]
	if !ok {
		buf = NewLogBuffer(s.bufferSize)
		s.buffers[componentID] = buf
	}
	s.mu.Unlock()
	buf.Push(entry)
}

func (s *LogStore) Snapshot(componentID string) []LogEntry {
	s.mu.RLock()
	buf, ok := s.buffers[componentID]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	return buf.Snapshot()
}

func (s *LogStore) Delete(componentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buffers, componentID)
}

// HandleComponentLogs subscribes to NATS subject aegis.logs.<component_id>,
// replays buffered lines (backlog), then streams live until client disconnects.
//
// A buffered channel decouples the NATS callback (which must return fast to
// avoid slow-consumer drops) from the socket write goroutine.
func HandleComponentLogs(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore, nc *nats.Conn, logStore *servicescomponent.LogStore) {
	log := logger.WithComponent("ComponentLogs").WithField("request_id", cmd.RequestID)

	var payload core.ComponentLogPathPayload
	raw, _ := json.Marshal(cmd.Payload)
	if err := json.Unmarshal(raw, &payload); err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   string(core.CommandComponentLogs),
			Status:    "error",
			ErrorCode: "INVALID_PAYLOAD",
			Message:   "failed to decode payload",
		})
		return
	}

	session, ok := sessionStore.GetSessionByIDApproximation(payload.SessionID)
	if !ok {
		session, ok = sessionStore.GetSessionByName(payload.SessionID)
	}
	if !ok {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   string(core.CommandComponentLogs),
			Status:    "error",
			ErrorCode: "SESSION_NOT_FOUND",
			Message:   "session not found: " + payload.SessionID,
		})
		return
	}

	registry := session.Registry
	comp, exists := registry.Get(payload.ComponentID)
	if !exists {
		comp, exists = registry.GetByName(session.ID, payload.ComponentID)
	}
	if !exists {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   string(core.CommandComponentLogs),
			Status:    "error",
			ErrorCode: "COMPONENT_NOT_FOUND",
			Message:   "component not found: " + payload.ComponentID,
		})
		return
	}

	subject := "aegis.logs." + comp.ID
	log.Infof("Streaming logs for component %s (%s) on subject %s", comp.Name, comp.ID, subject)

	core.WriteJSON(conn, core.Response{
		RequestID: cmd.RequestID,
		Command:   string(core.CommandComponentLogs),
		Status:    "ok",
		Message:   "streaming",
		Data: map[string]any{
			"component_id":   comp.ID,
			"component_name": comp.Name,
			"subject":        subject,
		},
	})

	// Buffered channel decouples the NATS callback from the socket writer.
	// 4096 entries gives plenty of headroom for burst log output without
	// blocking the NATS dispatcher goroutine.
	ch := make(chan LogEntry, 4096)
	done := make(chan struct{})

	// Subscribe BEFORE replaying backlog so no live messages are missed.
	var sub *nats.Subscription
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var entry LogEntry
		if jsonErr := json.Unmarshal(msg.Data, &entry); jsonErr != nil {
			entry = LogEntry{
				ComponentID:   comp.ID,
				ComponentName: comp.Name,
				Timestamp:     time.Now(),
				Level:         "raw",
				Message:       string(msg.Data),
			}
		}

		// Non-blocking send: if the channel is full the consumer is too slow
		// and we drop rather than block the NATS dispatcher.
		select {
		case ch <- entry:
		case <-done:
		default:
			log.Warnf("Log channel full — dropping message for component %s", comp.ID)
		}
	})
	if err != nil {
		log.Errorf("Failed to subscribe to %s: %v", subject, err)
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   string(core.CommandComponentLogs),
			Status:    "error",
			ErrorCode: "NATS_SUBSCRIBE_FAILED",
			Message:   err.Error(),
		})
		return
	}

	// Writer goroutine: drains the channel and writes to the CLI socket.
	// Runs independently so the read-loop below can detect disconnection
	// without racing with socket writes.
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		enc := json.NewEncoder(conn)

		// Replay backlog first.
		for _, entry := range logStore.Snapshot(comp.ID) {
			if err := enc.Encode(entry); err != nil {
				return
			}
		}

		// Then drain live messages.
		for {
			select {
			case entry, ok := <-ch:
				if !ok {
					return
				}
				if err := enc.Encode(entry); err != nil {
					return
				}
			case <-done:
				// Drain whatever is left in the channel before exiting.
				for {
					select {
					case entry := <-ch:
						_ = enc.Encode(entry)
					default:
						return
					}
				}
			}
		}
	}()

	// Block until the CLI closes the connection.
	buf := make([]byte, 1)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, readErr := conn.Read(buf)
		if readErr != nil {
			break
		}
	}

	close(done)
	_ = sub.Unsubscribe()
	close(ch)
	<-writerDone
	log.Infof("Log stream closed for component %s", comp.ID)
}
