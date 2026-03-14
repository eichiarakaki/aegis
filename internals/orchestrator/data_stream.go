package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/nats-io/nats.go"
)

// DataStreamHandshake is sent by the component when connecting to the data stream socket.
type DataStreamHandshake struct {
	ComponentID  string `json:"component_id"`
	SessionToken string `json:"session_token"`
}

// DataStreamHandshakeResponse is sent back after a successful handshake.
type DataStreamHandshakeResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message,omitempty"`
	Topics  []string `json:"topics,omitempty"`
}

// subscriber tracks a connected component and the topics it cares about.
type subscriber struct {
	componentID string
	topics      map[string]struct{} // full NATS topic strings
	ch          chan []byte
	// realtime=true  → ch is buffered; Deliver() is non-blocking (drop on full).
	// realtime=false → ch is unbuffered; Deliver() blocks for backpressure.
	realtime bool
}

// realtimeSubscriberBufSize is the number of messages that can be queued per
// subscriber before Deliver() starts dropping for a slow realtime component.
const realtimeSubscriberBufSize = 512

// DataStreamServer listens on a Unix socket and delivers NATS messages
// to connected components, filtered by each component's declared topics.
type DataStreamServer struct {
	session    *core.Session
	nc         *nats.Conn
	socketPath string
	listener   net.Listener
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	log        *logger.Logger

	realtime bool

	subsMu sync.RWMutex
	subs   []*subscriber
}

// NewDataStreamServer creates a DataStreamServer for the given session.
func NewDataStreamServer(session *core.Session, nc *nats.Conn) *DataStreamServer {
	socketPath := fmt.Sprintf("/tmp/aegis-data-stream-%s.sock", session.ID)
	return &DataStreamServer{
		session:    session,
		nc:         nc,
		socketPath: socketPath,
		log:        logger.WithComponent("DataStream").WithField("session_id", session.ID),
		realtime:   session.Mode == "realtime",
	}
}

// Start opens the Unix socket and begins accepting component connections.
func (s *DataStreamServer) Start(ctx context.Context) error {
	_ = os.Remove(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("data_stream: listen %s: %w", s.socketPath, err)
	}
	s.listener = ln

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.acceptLoop(runCtx)
	}()

	s.log.Infof("Data stream server listening on %s", s.socketPath)
	return nil
}

// Stop closes the listener and waits for all connections to drain.
func (s *DataStreamServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.wg.Wait()
	_ = os.Remove(s.socketPath)
	s.log.Infof("Data stream server stopped")
}

// Deliver sends data to every subscriber interested in natsTopic.
func (s *DataStreamServer) Deliver(natsTopic string, data []byte) {
	s.subsMu.RLock()
	defer s.subsMu.RUnlock()

	s.log.Debugf("Deliver: topic=%s subscribers=%d", natsTopic, len(s.subs))

	for _, sub := range s.subs {
		if _, ok := sub.topics[natsTopic]; !ok {
			continue
		}

		if sub.realtime {
			select {
			case sub.ch <- data:
			default:
				s.log.Warnf("Deliver: subscriber %s is too slow, dropping message for %s",
					sub.componentID, natsTopic)
			}
		} else {
			sub.ch <- data
		}
	}
}

func (s *DataStreamServer) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				s.log.Warnf("Accept error: %v", err)
				return
			}
		}

		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			s.handleConn(ctx, c)
		}(conn)
	}
}

func (s *DataStreamServer) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	log := s.log.WithField("remote_addr", conn.RemoteAddr().String())
	log.Debugf("Component connected to data stream")

	// Use a single bufio.Reader for the entire connection lifetime.
	// IMPORTANT: json.NewDecoder internally buffers reads — if we created a
	// new bufio.Reader just for the handshake and discarded it, bytes already
	// read into the decoder's internal buffer would be lost. By creating one
	// shared reader here and passing it to json.NewDecoder, any bytes the
	// decoder reads beyond the handshake JSON remain buffered in this reader
	// and are available for subsequent reads (none are expected from the
	// client after handshake, but this prevents accidental data loss).
	connReader := bufio.NewReader(conn)
	dec := json.NewDecoder(connReader)
	enc := json.NewEncoder(conn)

	// --- Handshake ---
	var hs DataStreamHandshake
	if err := dec.Decode(&hs); err != nil {
		log.Warnf("Failed to read handshake: %v", err)
		return
	}

	if hs.SessionToken != s.session.ID {
		log.Warnf("Invalid session_token: %s", hs.SessionToken)
		_ = enc.Encode(map[string]string{"status": "error", "message": "invalid session_token"})
		return
	}

	_, ok := s.session.Registry.Get(hs.ComponentID)
	if !ok {
		log.Warnf("Unknown component_id: %s", hs.ComponentID)
		_ = enc.Encode(map[string]string{"status": "error", "message": "unknown component_id"})
		return
	}

	componentTopics := s.topicsForComponent(hs.ComponentID)
	if len(componentTopics) == 0 {
		log.Warnf("Component %s has no topics", hs.ComponentID)
		_ = enc.Encode(map[string]string{"status": "error", "message": "no topics for component"})
		return
	}

	topicSet := make(map[string]struct{}, len(componentTopics))
	for _, t := range componentTopics {
		topicSet[t] = struct{}{}
	}

	var ch chan []byte
	if s.realtime {
		ch = make(chan []byte, realtimeSubscriberBufSize)
	} else {
		ch = make(chan []byte)
	}

	sub := &subscriber{
		componentID: hs.ComponentID,
		topics:      topicSet,
		ch:          ch,
		realtime:    s.realtime,
	}

	s.subsMu.Lock()
	s.subs = append(s.subs, sub)
	s.subsMu.Unlock()

	defer s.removeSub(sub)

	// Send handshake response via json.NewEncoder on the raw conn (not the
	// bufio.Reader — that is read-only). enc was created on conn directly.
	if err := enc.Encode(DataStreamHandshakeResponse{Status: "ok", Topics: componentTopics}); err != nil {
		log.Warnf("Failed to send handshake response: %v", err)
		return
	}

	log.Infof("Handshake OK — component=%s topics=%v", hs.ComponentID, componentTopics)

	// --- Write loop: forward messages to socket as newline-delimited JSON ---
	// Use a fresh bufio.Writer for outbound data — independent of the inbound
	// connReader above.
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()

	writer := bufio.NewWriterSize(conn, 64*1024)
	for {
		select {
		case <-connCtx.Done():
			return
		case data := <-sub.ch:
			if _, err := writer.Write(data); err != nil {
				log.Debugf("Write error: %v", err)
				return
			}
			if err := writer.WriteByte('\n'); err != nil {
				log.Debugf("Write error: %v", err)
				return
			}
			if err := writer.Flush(); err != nil {
				log.Debugf("Flush error: %v", err)
				return
			}
		}
	}
}

func (s *DataStreamServer) removeSub(sub *subscriber) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	for i, v := range s.subs {
		if v == sub {
			s.subs = append(s.subs[:i], s.subs[i+1:]...)
			return
		}
	}
}

// topicsForComponent returns the full NATS topic strings for a given component ID.
func (s *DataStreamServer) topicsForComponent(componentID string) []string {
	var topics []string

	s.session.WithRLock(func() {
		for topic, owners := range s.session.TopicOwners {
			for _, ownerID := range owners {
				if ownerID == componentID {
					topics = append(topics, fmt.Sprintf("aegis.%s.%s", s.session.ID, topic))
					break
				}
			}
		}
	})

	return topics
}
