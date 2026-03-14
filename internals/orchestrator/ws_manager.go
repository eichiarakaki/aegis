package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/gorilla/websocket"
)

const (
	binanceWSSpot    = "wss://stream.binance.com:9443"
	binanceWSFutures = "wss://fstream.binance.com"
	binanceWSCoinM   = "wss://dstream.binance.com"

	wsReconnectDelay    = 3 * time.Second
	wsMaxReconnectDelay = 60 * time.Second
	wsPingInterval      = 20 * time.Second
	wsReadTimeout       = 60 * time.Second
)

// Market identifies which Binance market a session targets.
type Market string

const (
	MarketSpot    Market = "spot"
	MarketFutures Market = "futures" // USD-M perpetual
	MarketCoinM   Market = "coin-m"  // COIN-M perpetual
)

func wsBaseURL(market Market) string {
	switch market {
	case MarketFutures:
		return binanceWSFutures
	case MarketCoinM:
		return binanceWSCoinM
	default:
		return binanceWSSpot
	}
}

// streamName returns the Binance WebSocket stream name for a given topic.
func streamName(dataType, symbol, timeframe string, market Market) (string, error) {
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
		speed := timeframe
		if speed == "" {
			speed = "100ms"
		}
		if speed != "100ms" && speed != "250ms" && speed != "500ms" {
			return "", fmt.Errorf("ws_manager: invalid orderBook speed %q (valid: 100ms, 250ms, 500ms)", speed)
		}
		return fmt.Sprintf("%s@depth20@%s", sym, speed), nil
	default:
		return "", fmt.Errorf("ws_manager: no WebSocket stream for data type %q", dataType)
	}
}

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
// publishes every incoming message directly via Publisher.
type WSManager struct {
	sessionID string
	market    Market
	subs      []wsSubscription
	log       *logger.Logger
	wg        sync.WaitGroup
}

func NewWSManager(sessionID string, market Market, subs []wsSubscription) *WSManager {
	return &WSManager{
		sessionID: sessionID,
		market:    market,
		subs:      subs,
		log:       logger.WithComponent("WSManager").WithField("session_id", sessionID),
	}
}

func (m *WSManager) Start(ctx context.Context) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.connectLoop(ctx)
	}()
}

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

	baseURL := wsBaseURL(m.market)
	u := fmt.Sprintf("%s/stream?streams=%s", baseURL, strings.Join(names, "/"))

	m.log.Infof("Connecting to %s (market=%s)", u, m.market)

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

	var writeMu sync.Mutex
	safeWrite := func(msgType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(msgType, data)
	}

	conn.SetPingHandler(func(appData string) error {
		m.log.Debugf("Received server ping — sending pong")
		return safeWrite(websocket.PongMessage, []byte(appData))
	})

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
				if err := safeWrite(websocket.PongMessage, nil); err != nil {
					m.log.Warnf("Keepalive pong failed: %v", err)
					return
				}
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

type combinedStreamWrapper struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

func (m *WSManager) dispatch(lookup map[string]*wsSubscription, raw []byte) {
	// DEBUG: confirm messages are arriving from Binance
	m.log.Debugf("dispatch: raw message received (%d bytes): %.200s", len(raw), string(raw))

	var wrapper combinedStreamWrapper
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		m.log.Warnf("Failed to unwrap combined message: %v", err)
		return
	}

	m.log.Debugf("dispatch: stream=%q", wrapper.Stream)

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
		m.log.Warnf("No subscription for stream %q (known: %v)", wrapper.Stream, func() []string {
			keys := make([]string, 0, len(lookup))
			for k := range lookup {
				keys = append(keys, k)
			}
			return keys
		}())
		return
	}

	ts, payload, err := sub.parseFn(wrapper.Data)
	if err != nil {
		m.log.Warnf("Parse error for stream %q: %v | data=%.200s", wrapper.Stream, err, string(wrapper.Data))
		return
	}

	m.log.Debugf("dispatch: parsed stream=%q ts=%d topic=%s payload_bytes=%d",
		wrapper.Stream, ts, sub.natsTopic, len(payload))

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
