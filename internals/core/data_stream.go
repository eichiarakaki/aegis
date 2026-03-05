package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

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
	Status string   `json:"status"`
	Topics []string `json:"topics"`
}

// DataStreamServer listens on a Unix socket and forwards NATS messages
// to connected components, filtered by each component's declared topics.
type DataStreamServer struct {
	session    *Session
	nc         *nats.Conn
	socketPath string
	listener   net.Listener
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	log        *logger.Logger
}

// NewDataStreamServer creates a DataStreamServer for the given session.
// The socket path must match what was sent in CONFIGURE.
func NewDataStreamServer(session *Session, nc *nats.Conn) *DataStreamServer {
	socketPath := fmt.Sprintf("/tmp/aegis-data-stream-%s.sock", session.ID)
	return &DataStreamServer{
		session:    session,
		nc:         nc,
		socketPath: socketPath,
		log:        logger.WithComponent("DataStream").WithField("session_id", session.ID),
	}
}

// Start opens the Unix socket and begins accepting component connections.
func (s *DataStreamServer) Start(ctx context.Context) error {
	// Remove stale socket file if it exists.
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

	// --- Handshake ---
	dec := json.NewDecoder(bufio.NewReader(conn))
	enc := json.NewEncoder(conn)

	var hs DataStreamHandshake
	if err := dec.Decode(&hs); err != nil {
		log.Warnf("Failed to read handshake: %v", err)
		return
	}

	// Validate component_id belongs to this session and token matches.
	if hs.SessionToken != s.session.ID {
		log.Warnf("Invalid session_token in handshake: %s", hs.SessionToken)
		_ = enc.Encode(map[string]string{"status": "error", "message": "invalid session_token"})
		return
	}

	comp, ok := s.session.Registry.Get(hs.ComponentID)
	if !ok {
		log.Warnf("Unknown component_id in handshake: %s", hs.ComponentID)
		_ = enc.Encode(map[string]string{"status": "error", "message": "unknown component_id"})
		return
	}

	// Resolve the NATS topics this component is subscribed to.
	// session.TopicOwners maps topic → []componentID (component-facing topic format).
	// We need to convert to full NATS topics: aegis.<sid>.<topic>
	componentTopics := s.topicsForComponent(hs.ComponentID)
	if len(componentTopics) == 0 {
		log.Warnf("Component %s has no topics in this session", hs.ComponentID)
		_ = enc.Encode(map[string]string{"status": "error", "message": "no topics for component"})
		return
	}

	resp := DataStreamHandshakeResponse{
		Status: "ok",
		Topics: componentTopics,
	}
	if err := enc.Encode(resp); err != nil {
		log.Warnf("Failed to send handshake response: %v", err)
		return
	}

	log.Infof("Handshake OK — component=%s topics=%v", comp.Name, componentTopics)

	// --- Subscribe to NATS and forward to socket ---
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()

	// One buffered channel per connection — NATS callbacks push here,
	// the write loop drains to the socket.
	msgCh := make(chan []byte, 256)

	var subs []*nats.Subscription
	for _, natsTopic := range componentTopics {
		natsTopic := natsTopic
		sub, err := s.nc.Subscribe(natsTopic, func(msg *nats.Msg) {
			select {
			case msgCh <- msg.Data:
			case <-connCtx.Done():
			default:
				// Channel full — drop message rather than block NATS callback.
				log.Warnf("msgCh full, dropping message on topic %s", natsTopic)
			}
		})
		if err != nil {
			log.Errorf("Failed to subscribe to %s: %v", natsTopic, err)
			continue
		}
		subs = append(subs, sub)
	}

	defer func() {
		for _, sub := range subs {
			_ = sub.Unsubscribe()
		}
	}()

	// Write loop: forward messages from NATS to the socket as JSON Lines.
	writer := bufio.NewWriter(conn)
	for {
		select {
		case <-connCtx.Done():
			return
		case <-ctx.Done():
			return
		case data := <-msgCh:
			// data is already a JSON envelope from the publisher.
			// Write it as a JSON Line (newline-delimited).
			if _, err := writer.Write(data); err != nil {
				log.Debugf("Write error (component disconnected): %v", err)
				return
			}
			if err := writer.WriteByte('\n'); err != nil {
				log.Debugf("Write error (component disconnected): %v", err)
				return
			}
			if err := writer.Flush(); err != nil {
				log.Debugf("Flush error (component disconnected): %v", err)
				return
			}
		}
	}
}

// topicsForComponent returns the full NATS topic strings for a given component ID.
func (s *DataStreamServer) topicsForComponent(componentID string) []string {
	s.session.mu.RLock()
	defer s.session.mu.RUnlock()

	var topics []string
	for topic, owners := range s.session.TopicOwners {
		for _, ownerID := range owners {
			if ownerID == componentID {
				// Convert component-facing topic to full NATS subject
				// Example:
				//   component topic: "klines.BTCUSDT.1m"
				//   NATS subject:    "aegis.<session_id>.klines.BTCUSDT.1m"
				natsTopic := fmt.Sprintf("aegis.%s.%s", s.session.ID, topic)
				topics = append(topics, natsTopic)
				break // no need to check other owners for this topic
			}
		}
	}
	return topics
}
