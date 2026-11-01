package bybit

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"orderbook/internal/exchange"

	"github.com/gorilla/websocket"
)

// FuturesExchange implements the Exchange interface for Bybit Futures
type FuturesExchange struct {
	symbol           string
	wsURL            string
	wsConn           *websocket.Conn
	updateChan       chan *exchange.DepthUpdate
	done             chan struct{}
	ctx              context.Context
	cancel           context.CancelFunc
	health           atomic.Value // stores exchange.HealthStatus
	snapshotReceived bool
	lastUpdateID     int64
	snapshot         *exchange.Snapshot
	snapshotMu       sync.Mutex
}

// Config holds configuration for Bybit Futures exchange
type Config struct {
	Symbol string
}

// NewFuturesExchange creates a new Bybit Futures exchange instance
func NewFuturesExchange(config Config) *FuturesExchange {
	ctx, cancel := context.WithCancel(context.Background())

	wsURL := "wss://stream.bybit.com/v5/public/linear"

	ex := &FuturesExchange{
		symbol:     config.Symbol,
		wsURL:      wsURL,
		updateChan: make(chan *exchange.DepthUpdate, 1000),
		done:       make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
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
func (e *FuturesExchange) GetName() exchange.ExchangeName {
	return exchange.Bybitf
}

// GetSymbol returns the trading symbol
func (e *FuturesExchange) GetSymbol() string {
	return e.symbol
}

// Connect establishes WebSocket connection to Bybit Futures
func (e *FuturesExchange) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, e.wsURL, nil)
	if err != nil {
		e.incrementErrorCount()
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	e.wsConn = conn
	e.updateConnectionStatus(true)
	log.Printf("[%s] WebSocket connected successfully", e.GetName())

	// Subscribe to orderbook stream (using depth 200 for full orderbook)
	subscribeMsg := SubscribeMessage{
		Op:   "subscribe",
		Args: []string{fmt.Sprintf("orderbook.1000.%s", e.symbol)},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		e.incrementErrorCount()
		conn.Close()
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("[%s] Subscribed to orderbook.1000.%s", e.GetName(), e.symbol)

	go e.readMessages()

	return nil
}

// Close closes the WebSocket connection
func (e *FuturesExchange) Close() error {
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

// GetSnapshot fetches the initial orderbook snapshot via WebSocket
// For Bybit, the first message received will be a snapshot
func (e *FuturesExchange) GetSnapshot(ctx context.Context) (*exchange.Snapshot, error) {
	log.Printf("[%s] Waiting for orderbook snapshot from WebSocket...", e.GetName())

	// Wait for the first snapshot message from the WebSocket
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			return nil, fmt.Errorf("timeout waiting for snapshot")
		default:
			e.snapshotMu.Lock()
			snap := e.snapshot
			e.snapshotMu.Unlock()

			if snap != nil {
				return snap, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Updates returns a channel that receives depth updates
func (e *FuturesExchange) Updates() <-chan *exchange.DepthUpdate {
	return e.updateChan
}

// IsConnected checks if the WebSocket connection is active
func (e *FuturesExchange) IsConnected() bool {
	return e.Health().Connected
}

// Health returns connection health information
func (e *FuturesExchange) Health() exchange.HealthStatus {
	if status, ok := e.health.Load().(exchange.HealthStatus); ok {
		return status
	}
	return exchange.HealthStatus{}
}

// readMessages continuously reads WebSocket messages
func (e *FuturesExchange) readMessages() {
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
			var msg WSMessage
			if err := e.wsConn.ReadJSON(&msg); err != nil {
				e.incrementErrorCount()
				log.Printf("[%s] WebSocket read error: %v", e.GetName(), err)
				return
			}

			// Skip non-orderbook messages
			if msg.Topic == "" || msg.Data.Symbol == "" {
				continue
			}

			e.incrementMessageCount()
			e.updateLastPing()

			// Handle initial snapshot
			if msg.Type == "snapshot" && !e.snapshotReceived {
				e.storeSnapshot(&msg)
				e.snapshotReceived = true
			}

			canonicalUpdate := e.convertDepthUpdate(&msg)

			select {
			case e.updateChan <- canonicalUpdate:
			case <-e.ctx.Done():
				return
			case <-e.done:
				return
			}
		}
	}
}

// storeSnapshot converts and stores the initial snapshot
func (e *FuturesExchange) storeSnapshot(msg *WSMessage) {
	bids := make([]exchange.PriceLevel, len(msg.Data.Bids))
	for i, bid := range msg.Data.Bids {
		bids[i] = exchange.PriceLevel{
			Price:    bid[0],
			Quantity: bid[1],
		}
	}

	asks := make([]exchange.PriceLevel, len(msg.Data.Asks))
	for i, ask := range msg.Data.Asks {
		asks[i] = exchange.PriceLevel{
			Price:    ask[0],
			Quantity: ask[1],
		}
	}

	snapshot := &exchange.Snapshot{
		Exchange:     e.GetName(),
		Symbol:       msg.Data.Symbol,
		LastUpdateID: msg.Data.UpdateID,
		Bids:         bids,
		Asks:         asks,
		Timestamp:    time.UnixMilli(msg.TS),
	}

	e.snapshotMu.Lock()
	e.snapshot = snapshot
	e.lastUpdateID = msg.Data.UpdateID
	e.snapshotMu.Unlock()
}

// convertDepthUpdate converts Bybit depth update to canonical format
func (e *FuturesExchange) convertDepthUpdate(msg *WSMessage) *exchange.DepthUpdate {
	bids := make([]exchange.PriceLevel, len(msg.Data.Bids))
	for i, bid := range msg.Data.Bids {
		bids[i] = exchange.PriceLevel{
			Price:    bid[0],
			Quantity: bid[1],
		}
	}

	asks := make([]exchange.PriceLevel, len(msg.Data.Asks))
	for i, ask := range msg.Data.Asks {
		asks[i] = exchange.PriceLevel{
			Price:    ask[0],
			Quantity: ask[1],
		}
	}

	// Bybit's `seq` is a cross-sequence shared by every symbol and jumps by
	// thousands between messages. Continuity must use `u` (update ID), which
	// increments by one per orderbook message for this symbol.
	prevUpdateID := e.lastUpdateID
	e.lastUpdateID = msg.Data.UpdateID

	return &exchange.DepthUpdate{
		Exchange:      e.GetName(),
		Symbol:        msg.Data.Symbol,
		EventTime:     time.UnixMilli(msg.TS),
		FirstUpdateID: msg.Data.UpdateID,
		FinalUpdateID: msg.Data.UpdateID,
		PrevUpdateID:  prevUpdateID,
		Bids:          bids,
		Asks:          asks,
	}
}

// updateConnectionStatus updates the connection status in health
func (e *FuturesExchange) updateConnectionStatus(connected bool) {
	status := e.Health()
	status.Connected = connected
	if !connected {
		now := time.Now()
		status.ReconnectTime = &now
	}
	e.health.Store(status)
}

// incrementMessageCount increments the message count in health
func (e *FuturesExchange) incrementMessageCount() {
	status := e.Health()
	status.MessageCount++
	e.health.Store(status)
}

// incrementErrorCount increments the error count in health
func (e *FuturesExchange) incrementErrorCount() {
	status := e.Health()
	status.ErrorCount++
	e.health.Store(status)
}

// updateLastPing updates the last ping time in health
func (e *FuturesExchange) updateLastPing() {
	status := e.Health()
	status.LastPing = time.Now()
	e.health.Store(status)
}
