package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/gorilla/websocket"
)

const (
	binanceWSBaseURL    = "wss://fstream.binance.com"
	wsReconnectDelay    = 3 * time.Second
	wsMaxReconnectDelay = 60 * time.Second
	wsPingInterval      = 20 * time.Second
	wsReadTimeout       = 60 * time.Second
)

// streamName returns the Binance WebSocket stream name for a given topic.
func streamName(dataType, symbol, timeframe string) (string, error) {
	sym := strings.ToLower(symbol)
	switch dataType {
	case "klines":
		if timeframe == "" {
			return "", fmt.Errorf("ws_manager: klines requires a timeframe")
		}
		return fmt.Sprintf("%s@kline_%s", sym, timeframe), nil
	case "aggTrades":
		return fmt.Sprintf("%s@aggTrade", sym), nil
	case "trades":
		return fmt.Sprintf("%s@trade", sym), nil
	case "orderBook":
		return fmt.Sprintf("%s@depth20@100ms", sym), nil
	default:
		return "", fmt.Errorf("ws_manager: no WebSocket stream for data type %q", dataType)
	}
}

// wsSubscription represents a single stream-to-publisher binding.
// All subscriptions in realtime mode are clockless — rows are published
// immediately without going through a GlobalClock barrier.
type wsSubscription struct {
	streamName string
	dataType   string
	priority   int
	natsTopic  string
	parseFn    WSParseFunc
	pub        *Publisher
	sessionID  string
}

// WSManager opens a single Binance combined-stream WebSocket connection and
// publishes every incoming message directly to NATS via Publisher.
// There is no clock, no buffer, and no ordering guarantee between streams —
// rows are delivered as fast as Binance sends them.
type WSManager struct {
	sessionID string
	subs      []wsSubscription
	log       *logger.Logger
	wg        sync.WaitGroup
}

// NewWSManager creates a WSManager for the given subscriptions.
func NewWSManager(sessionID string, subs []wsSubscription) *WSManager {
	return &WSManager{
		sessionID: sessionID,
		subs:      subs,
		log:       logger.WithComponent("WSManager").WithField("session_id", sessionID),
	}
}

// Start launches the connection loop in a background goroutine.
func (m *WSManager) Start(ctx context.Context) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.connectLoop(ctx)
	}()
}

// Stop waits for the connection goroutine to exit.
func (m *WSManager) Stop() {
	m.wg.Wait()
}

func (m *WSManager) connectLoop(ctx context.Context) {
	delay := wsReconnectDelay
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := m.connect(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			m.log.Warnf("WebSocket disconnected: %v — reconnecting in %s", err, delay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			delay = minDuration(delay*2, wsMaxReconnectDelay)
			continue
		}
		return
	}
}

func (m *WSManager) connect(ctx context.Context) error {
	names := make([]string, len(m.subs))
	for i, sub := range m.subs {
		names[i] = sub.streamName
	}
	u := fmt.Sprintf("%s/stream?streams=%s", binanceWSBaseURL, url.QueryEscape(strings.Join(names, "/")))

	m.log.Infof("Connecting to %s", u)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	m.log.Infof("Connected — %d streams active", len(m.subs))

	lookup := make(map[string]*wsSubscription, len(m.subs))
	for i := range m.subs {
		lookup[m.subs[i].streamName] = &m.subs[i]
	}

	pingCtx, cancelPing := context.WithCancel(ctx)
	defer cancelPing()
	go func() {
		ticker := time.NewTicker(wsPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return
			case <-ticker.C:
				_ = conn.WriteMessage(websocket.PingMessage, nil)
			}
		}
	}()

	for {
		if ctx.Err() != nil {
			return nil
		}
		conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}
		m.dispatch(lookup, raw)
	}
}

// combinedStreamWrapper is the envelope Binance uses for combined streams.
type combinedStreamWrapper struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

// dispatch parses a combined-stream message and publishes it immediately.
func (m *WSManager) dispatch(lookup map[string]*wsSubscription, raw []byte) {
	var wrapper combinedStreamWrapper
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		m.log.Warnf("Failed to unwrap combined message: %v", err)
		return
	}

	sub := lookup[wrapper.Stream]
	if sub == nil {
		for k, v := range lookup {
			if strings.EqualFold(k, wrapper.Stream) {
				sub = v
				break
			}
		}
	}
	if sub == nil {
		m.log.Warnf("No subscription for stream %q", wrapper.Stream)
		return
	}

	ts, payload, err := sub.parseFn(wrapper.Data)
	if err != nil {
		m.log.Warnf("Parse error for stream %q: %v", wrapper.Stream, err)
		return
	}

	row := RawRow{
		Timestamp: ts,
		DataType:  sub.dataType,
		Priority:  sub.priority,
		Topic:     sub.natsTopic,
		Payload:   payload,
	}

	if err := sub.pub.Publish(sub.sessionID, row); err != nil {
		m.log.Warnf("Publish error for %q: %v", sub.natsTopic, err)
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
