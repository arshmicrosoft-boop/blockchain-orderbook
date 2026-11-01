package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"orderbook/internal/exchange"
)

// FuturesExchange implements the Exchange interface for Hyperliquid
type FuturesExchange struct {
	symbol     string
	wsURL      string
	restURL    string
	wsConn     *websocket.Conn
	updateChan chan *exchange.DepthUpdate
	done       chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	health     atomic.Value // stores exchange.HealthStatus
}

// Config holds configuration for Hyperliquid exchange
type Config struct {
	Symbol string
}

// NewFuturesExchange creates a new Hyperliquid exchange instance
func NewFuturesExchange(config Config) *FuturesExchange {
	ctx, cancel := context.WithCancel(context.Background())

	// Convert XXXUSDT to XXX for Hyperliquid (e.g., BTCUSDT -> BTC)
	symbol := strings.TrimSuffix(config.Symbol, "USDT")

	ex := &FuturesExchange{
		symbol:     symbol,
		wsURL:      "wss://api.hyperliquid.xyz/ws",
		restURL:    "https://api.hyperliquid.xyz/info",
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
	return exchange.Hyperliquidf
}

// GetSymbol returns the trading symbol
func (e *FuturesExchange) GetSymbol() string {
	return e.symbol
}

// Connect establishes WebSocket connection to Hyperliquid
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

	// Subscribe to L2 book updates
	subscription := SubscriptionMessage{
		Method: "subscribe",
		Subscription: map[string]interface{}{
			"type": "l2Book",
			"coin": e.symbol,
		},
	}

	if err := conn.WriteJSON(subscription); err != nil {
		e.incrementErrorCount()
		return fmt.Errorf("failed to send subscription: %w", err)
	}

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

// GetSnapshot fetches the initial orderbook snapshot via REST API
func (e *FuturesExchange) GetSnapshot(ctx context.Context) (*exchange.Snapshot, error) {
	log.Printf("[%s] Fetching orderbook snapshot...", e.GetName())

	requestBody := map[string]interface{}{
		"type": "l2Book",
		"coin": e.symbol,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.restURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		e.incrementErrorCount()
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	defer resp.Body.Close()

	var hyperliquidSnapshot L2BookResponse
	if err := json.NewDecoder(resp.Body).Decode(&hyperliquidSnapshot); err != nil {
		e.incrementErrorCount()
		return nil, fmt.Errorf("failed to decode snapshot: %w", err)
	}

	snapshot := e.convertSnapshot(&hyperliquidSnapshot)
	return snapshot, nil
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

			e.incrementMessageCount()
			e.updateLastPing()

			// Handle subscription response
			if msg.Channel == "subscriptionResponse" {
				continue
			}

			// Handle L2 book updates
			if msg.Channel == "l2Book" {
				var bookData WsBook
				dataBytes, err := json.Marshal(msg.Data)
				if err != nil {
					log.Printf("[%s] Error marshalling book data: %v", e.GetName(), err)
					continue
				}

				if err := json.Unmarshal(dataBytes, &bookData); err != nil {
					log.Printf("[%s] Error unmarshalling book data: %v", e.GetName(), err)
					continue
				}

				canonicalUpdate := e.convertDepthUpdate(&bookData)

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
}

// convertSnapshot converts Hyperliquid snapshot to canonical format
func (e *FuturesExchange) convertSnapshot(snapshot *L2BookResponse) *exchange.Snapshot {
	bids := make([]exchange.PriceLevel, len(snapshot.Levels[0]))
	for i, bid := range snapshot.Levels[0] {
		bids[i] = exchange.PriceLevel{
			Price:    bid.Px,
			Quantity: bid.Sz,
		}
	}

	asks := make([]exchange.PriceLevel, len(snapshot.Levels[1]))
	for i, ask := range snapshot.Levels[1] {
		asks[i] = exchange.PriceLevel{
			Price:    ask.Px,
			Quantity: ask.Sz,
		}
	}

	return &exchange.Snapshot{
		Exchange:     e.GetName(),
		Symbol:       e.symbol,
		LastUpdateID: snapshot.Time, // Use timestamp as update ID
		Bids:         bids,
		Asks:         asks,
		Timestamp:    time.UnixMilli(snapshot.Time),
	}
}

// convertDepthUpdate converts Hyperliquid book update to canonical format
func (e *FuturesExchange) convertDepthUpdate(update *WsBook) *exchange.DepthUpdate {
	bids := make([]exchange.PriceLevel, len(update.Levels[0]))
	for i, bid := range update.Levels[0] {
		bids[i] = exchange.PriceLevel{
			Price:    bid.Px,
			Quantity: bid.Sz,
		}
	}

	asks := make([]exchange.PriceLevel, len(update.Levels[1]))
	for i, ask := range update.Levels[1] {
		asks[i] = exchange.PriceLevel{
			Price:    ask.Px,
			Quantity: ask.Sz,
		}
	}

	return &exchange.DepthUpdate{
		Exchange:      e.GetName(),
		Symbol:        update.Coin,
		EventTime:     time.UnixMilli(update.Time),
		FirstUpdateID: 0,
		FinalUpdateID: 0,
		PrevUpdateID:  0,
		Bids:          bids,
		Asks:          asks,
		// Hyperliquid's l2Book stream sends a complete book every time, and
		// its timestamps are not a usable sequence number. Treating these as
		// deltas left the book permanently out of sync.
		IsSnapshot: true,
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