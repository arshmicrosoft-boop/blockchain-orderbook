package bingx

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"orderbook/internal/exchange"
)

const (
	wsURL = "wss://open-api-ws.bingx.com/market"
)

// SpotExchange implements the Exchange interface for BingX Spot
type SpotExchange struct {
	symbol         string
	bingxSymbol    string // BingX format (e.g., BTC-USDT)
	wsConn         *websocket.Conn
	updateChan     chan *exchange.DepthUpdate
	done           chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
	health         atomic.Value
	snapshotMutex  sync.Mutex
	snapshot       *exchange.Snapshot
	snapshotReady  chan struct{}
	hasSnapshot    bool
	lastUpdateID   int64
}

// NewSpotExchange creates a new BingX Spot exchange instance
func NewSpotExchange(config Config) *SpotExchange {
	ctx, cancel := context.WithCancel(context.Background())

	bingxSymbol := convertToBingXSymbol(config.Symbol)

	ex := &SpotExchange{
		symbol:        config.Symbol,
		bingxSymbol:   bingxSymbol,
		updateChan:    make(chan *exchange.DepthUpdate, 1000),
		done:          make(chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
		snapshotReady: make(chan struct{}),
		hasSnapshot:   false,
	}

	ex.health.Store(exchange.HealthStatus{
		Connected:    false,
		LastPing:     time.Time{},
		MessageCount: 0,
		ErrorCount:   0,
	})

	return ex
}

// GetName returns the exchange name
func (e *SpotExchange) GetName() exchange.ExchangeName {
	return exchange.BingX
}

// GetSymbol returns the trading symbol
func (e *SpotExchange) GetSymbol() string {
	return e.symbol
}

// Connect establishes WebSocket connection to BingX Spot
func (e *SpotExchange) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Add gzip compression support
	header := map[string][]string{
		"Accept-Encoding": {"gzip"},
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		e.incrementErrorCount()
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	e.wsConn = conn
	e.updateConnectionStatus(true)
	log.Printf("[%s] WebSocket connected successfully", e.GetName())

	// Subscribe to incremental depth
	subMsg := SubscriptionMessage{
		ID:       uuid.New().String(),
		ReqType:  "sub",
		DataType: fmt.Sprintf("%s@incrDepth", e.bingxSymbol),
	}

	if err := conn.WriteJSON(subMsg); err != nil {
		e.incrementErrorCount()
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("[%s] Subscribed to %s", e.GetName(), subMsg.DataType)

	go e.readMessages()
	go e.pingLoop()

	return nil
}

// Close closes the WebSocket connection
func (e *SpotExchange) Close() error {
	if e.cancel != nil {
		e.cancel()
	}

	if e.wsConn != nil {
		select {
		case <-e.done:
		default:
			close(e.done)
		}

		err := e.wsConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Printf("[%s] Error sending close message: %v", e.GetName(), err)
		}

		select {
		case <-time.After(time.Second):
		}

		e.updateConnectionStatus(false)
		return e.wsConn.Close()
	}
	return nil
}

// GetSnapshot waits for and returns the initial orderbook snapshot from WebSocket
func (e *SpotExchange) GetSnapshot(ctx context.Context) (*exchange.Snapshot, error) {
	log.Printf("[%s] Waiting for initial snapshot from WebSocket...", e.GetName())

	select {
	case <-e.snapshotReady:
		e.snapshotMutex.Lock()
		snapshot := e.snapshot
		e.snapshotMutex.Unlock()
		return snapshot, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for snapshot")
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for snapshot")
	}
}

// Updates returns a channel that receives depth updates
func (e *SpotExchange) Updates() <-chan *exchange.DepthUpdate {
	return e.updateChan
}

// IsConnected checks if the WebSocket connection is active
func (e *SpotExchange) IsConnected() bool {
	return e.Health().Connected
}

// Health returns connection health information
func (e *SpotExchange) Health() exchange.HealthStatus {
	if status, ok := e.health.Load().(exchange.HealthStatus); ok {
		return status
	}
	return exchange.HealthStatus{}
}

// pingLoop sends periodic pings (not needed for BingX, they send pings to us)
// But we keep the goroutine structure for consistency
func (e *SpotExchange) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-e.done:
			return
		case <-ticker.C:
			// BingX sends pings to us, we just respond with pong
			// This is just a keepalive check
			continue
		}
	}
}

// readMessages continuously reads WebSocket messages
func (e *SpotExchange) readMessages() {
	defer close(e.updateChan)
	defer e.updateConnectionStatus(false)

	for {
		select {
		case <-e.ctx.Done():
			log.Printf("[%s] Context cancelled, stopping message reading", e.GetName())
			return
		case <-e.done:
			return
		default:
			messageType, message, err := e.wsConn.ReadMessage()
			if err != nil {
				e.incrementErrorCount()
				log.Printf("[%s] WebSocket read error: %v", e.GetName(), err)
				return
			}

			if err := e.handleMessage(messageType, message); err != nil {
				log.Printf("[%s] Error handling message: %v", e.GetName(), err)
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages (text or binary/gzip)
func (e *SpotExchange) handleMessage(messageType int, message []byte) error {
	var decodedMsg string

	if messageType == websocket.TextMessage {
		decodedMsg = string(message)
	} else if messageType == websocket.BinaryMessage {
		// Decompress gzip
		decoded, err := decodeGzip(message)
		if err != nil {
			e.incrementErrorCount()
			return fmt.Errorf("failed to decode gzip: %w", err)
		}
		decodedMsg = decoded
	} else {
		return nil
	}

	// Handle ping/pong
	if strings.Contains(decodedMsg, "ping") || decodedMsg == "ping" {
		if err := e.wsConn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
			log.Printf("[%s] Failed to send pong: %v", e.GetName(), err)
		}
		return nil
	}

	// Parse JSON message
	var msg WSMessage
	if err := json.Unmarshal([]byte(decodedMsg), &msg); err != nil {
		// Might be a non-JSON message like "pong", ignore
		return nil
	}

	// Check for error response
	if msg.Code != 0 && msg.Msg != "" {
		return fmt.Errorf("BingX error: code=%d, msg=%s", msg.Code, msg.Msg)
	}

	// Handle depth data
	if msg.Data.Action == "all" {
		// This is the initial snapshot
		e.handleSnapshot(&msg)
	} else if msg.Data.Action == "update" {
		// This is an incremental update
		e.handleUpdate(&msg)
	}

	e.incrementMessageCount()
	e.updateLastPing()

	return nil
}

// handleSnapshot processes the initial full depth snapshot
func (e *SpotExchange) handleSnapshot(msg *WSMessage) {
	e.snapshotMutex.Lock()
	defer e.snapshotMutex.Unlock()

	if e.hasSnapshot {
		// Already have snapshot, treat as update
		return
	}

	snapshot := e.convertSnapshot(&msg.Data)
	e.snapshot = snapshot
	e.hasSnapshot = true
	e.lastUpdateID = snapshot.LastUpdateID

	log.Printf("[%s] Received initial snapshot with lastUpdateId=%d, bids=%d, asks=%d",
		e.GetName(), snapshot.LastUpdateID, len(snapshot.Bids), len(snapshot.Asks))

	// Signal that snapshot is ready
	select {
	case <-e.snapshotReady:
	default:
		close(e.snapshotReady)
	}
}

// handleUpdate processes incremental depth updates
func (e *SpotExchange) handleUpdate(msg *WSMessage) {
	canonicalUpdate := e.convertDepthUpdate(&msg.Data)

	select {
	case e.updateChan <- canonicalUpdate:
	case <-e.ctx.Done():
		return
	case <-e.done:
		return
	}
}

// convertSnapshot converts BingX snapshot to canonical format
func (e *SpotExchange) convertSnapshot(data *DepthData) *exchange.Snapshot {
	bids := make([]exchange.PriceLevel, 0, len(data.Bids))
	for price, quantity := range data.Bids {
		bids = append(bids, exchange.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	asks := make([]exchange.PriceLevel, 0, len(data.Asks))
	for price, quantity := range data.Asks {
		asks = append(asks, exchange.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	return &exchange.Snapshot{
		Exchange:     e.GetName(),
		Symbol:       e.symbol,
		LastUpdateID: data.LastUpdateID,
		Bids:         bids,
		Asks:         asks,
		Timestamp:    time.Now(),
	}
}

// convertDepthUpdate converts BingX depth update to canonical format
func (e *SpotExchange) convertDepthUpdate(data *DepthData) *exchange.DepthUpdate {
	prevUpdateID := e.lastUpdateID
	e.lastUpdateID = data.LastUpdateID
	bids := make([]exchange.PriceLevel, 0, len(data.Bids))
	for price, quantity := range data.Bids {
		bids = append(bids, exchange.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	asks := make([]exchange.PriceLevel, 0, len(data.Asks))
	for price, quantity := range data.Asks {
		asks = append(asks, exchange.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	return &exchange.DepthUpdate{
		Exchange:      e.GetName(),
		Symbol:        e.symbol,
		EventTime:     time.Now(),
		FirstUpdateID: data.LastUpdateID,
		FinalUpdateID: data.LastUpdateID,
		PrevUpdateID:  prevUpdateID,
		Bids:          bids,
		Asks:          asks,
	}
}

// decodeGzip decompresses gzip-encoded data
func decodeGzip(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()

	decodedMsg, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(decodedMsg), nil
}

// convertToBingXSymbol converts various symbol formats to BingX format
// Examples: BTCUSDT -> BTC-USDT, BTC-USDT -> BTC-USDT
func convertToBingXSymbol(symbol string) string {
	if strings.Contains(symbol, "-") {
		return strings.ToUpper(symbol)
	}

	symbol = strings.ToUpper(symbol)

	if strings.HasSuffix(symbol, "USDT") {
		base := strings.TrimSuffix(symbol, "USDT")
		return fmt.Sprintf("%s-USDT", base)
	}

	if strings.HasSuffix(symbol, "USD") && !strings.HasSuffix(symbol, "USDT") {
		base := strings.TrimSuffix(symbol, "USD")
		return fmt.Sprintf("%s-USD", base)
	}

	if strings.HasSuffix(symbol, "USDC") {
		base := strings.TrimSuffix(symbol, "USDC")
		return fmt.Sprintf("%s-USDC", base)
	}

	log.Printf("[BingX] Warning: Could not convert symbol %s to BingX format, using as-is", symbol)
	return symbol
}

// updateConnectionStatus updates the connection status in health
func (e *SpotExchange) updateConnectionStatus(connected bool) {
	status := e.Health()
	status.Connected = connected
	if !connected {
		now := time.Now()
		status.ReconnectTime = &now
	}
	e.health.Store(status)
}

// incrementMessageCount increments the message count in health
func (e *SpotExchange) incrementMessageCount() {
	status := e.Health()
	status.MessageCount++
	e.health.Store(status)
}

// incrementErrorCount increments the error count in health
func (e *SpotExchange) incrementErrorCount() {
	status := e.Health()
	status.ErrorCount++
	e.health.Store(status)
}

// updateLastPing updates the last ping time in health
func (e *SpotExchange) updateLastPing() {
	status := e.Health()
	status.LastPing = time.Now()
	e.health.Store(status)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
