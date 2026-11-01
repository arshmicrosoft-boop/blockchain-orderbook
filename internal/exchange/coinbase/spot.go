package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"orderbook/internal/exchange"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

// SpotExchange implements the Exchange interface for Coinbase Spot
type SpotExchange struct {
	symbol           string
	wsURL            string
	wsConn           *websocket.Conn
	updateChan       chan *exchange.DepthUpdate
	done             chan struct{}
	ctx              context.Context
	cancel           context.CancelFunc
	health           atomic.Value
	snapshot         *exchange.Snapshot
	snapshotMu       sync.Mutex
}

// NewSpotExchange creates a new Coinbase Spot exchange instance
func NewSpotExchange(config Config) *SpotExchange {
	ctx, cancel := context.WithCancel(context.Background())

	wsURL := "wss://advanced-trade-ws.coinbase.com"

	coinbaseSymbol := convertToCoinbaseSymbol(config.Symbol)

	ex := &SpotExchange{
		symbol:     coinbaseSymbol,
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
func (e *SpotExchange) GetName() exchange.ExchangeName {
	return exchange.Coinbase
}

// GetSymbol returns the trading symbol
func (e *SpotExchange) GetSymbol() string {
	return e.symbol
}

// Connect establishes WebSocket connection to Coinbase
func (e *SpotExchange) Connect(ctx context.Context) error {
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

	subscribeMsg := SubscribeRequest{
		Type:       "subscribe",
		ProductIDs: []string{e.symbol},
		Channel:    "level2",
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		e.incrementErrorCount()
		conn.Close()
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("[%s] Subscribed to level2 channel for %s", e.GetName(), e.symbol)

	go e.readMessages()

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

// GetSnapshot fetches the initial orderbook snapshot via WebSocket
func (e *SpotExchange) GetSnapshot(ctx context.Context) (*exchange.Snapshot, error) {
	log.Printf("[%s] Waiting for orderbook snapshot from WebSocket...", e.GetName())

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
			_, message, err := e.wsConn.ReadMessage()
			if err != nil {
				e.incrementErrorCount()
				log.Printf("[%s] WebSocket read error: %v", e.GetName(), err)
				return
			}

			var msg WSMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			if msg.Channel != "l2_data" || len(msg.Events) == 0 {
				continue
			}

			e.incrementMessageCount()
			e.updateLastPing()

			event := msg.Events[0]

			// Coinbase resends a snapshot after a resubscribe. Refresh the
			// stored snapshot and replace the book rather than ignoring it.
			if event.Type == "snapshot" {
				e.storeSnapshot(&event)
			}

			if event.Type == "snapshot" || event.Type == "update" {
				canonicalUpdate := e.convertDepthUpdate(&event)

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

// storeSnapshot converts and stores the initial snapshot
func (e *SpotExchange) storeSnapshot(event *Event) {
	var allBids, allAsks []exchange.PriceLevel

	for _, update := range event.Updates {
		if update.NewQuantity == "0" {
			continue
		}

		priceLevel := exchange.PriceLevel{
			Price:    update.PriceLevel,
			Quantity: update.NewQuantity,
		}

		if update.Side == "bid" {
			allBids = append(allBids, priceLevel)
		} else if update.Side == "ask" || update.Side == "offer" {
			allAsks = append(allAsks, priceLevel)
		}
	}

	filteredBids, filteredAsks := filterSnapshotByDistance(allBids, allAsks, 0.50)

	snapshot := &exchange.Snapshot{
		Exchange:     e.GetName(),
		Symbol:       event.ProductID,
		LastUpdateID: 0,
		Bids:         filteredBids,
		Asks:         filteredAsks,
		Timestamp:    time.Now(),
	}

	e.snapshotMu.Lock()
	e.snapshot = snapshot
	e.snapshotMu.Unlock()
}

// filterSnapshotByDistance filters bids/asks to keep only those within a certain percentage of the mid price
func filterSnapshotByDistance(bids, asks []exchange.PriceLevel, maxDistancePct float64) ([]exchange.PriceLevel, []exchange.PriceLevel) {
	if len(bids) == 0 || len(asks) == 0 {
		return bids, asks
	}

	var bestBid, bestAsk decimal.Decimal
	for _, bid := range bids {
		price, err := decimal.NewFromString(bid.Price)
		if err != nil {
			continue
		}
		if bestBid.IsZero() || price.GreaterThan(bestBid) {
			bestBid = price
		}
	}

	for _, ask := range asks {
		price, err := decimal.NewFromString(ask.Price)
		if err != nil {
			continue
		}
		if bestAsk.IsZero() || price.LessThan(bestAsk) {
			bestAsk = price
		}
	}

	if bestBid.IsZero() || bestAsk.IsZero() {
		return bids, asks
	}

	midPrice := bestBid.Add(bestAsk).Div(decimal.NewFromInt(2))
	maxDistance := midPrice.Mul(decimal.NewFromFloat(maxDistancePct))

	filteredBids := make([]exchange.PriceLevel, 0, len(bids))
	for _, bid := range bids {
		price, err := decimal.NewFromString(bid.Price)
		if err != nil {
			continue
		}
		distance := midPrice.Sub(price)
		if distance.LessThanOrEqual(maxDistance) {
			filteredBids = append(filteredBids, bid)
		}
	}

	filteredAsks := make([]exchange.PriceLevel, 0, len(asks))
	for _, ask := range asks {
		price, err := decimal.NewFromString(ask.Price)
		if err != nil {
			continue
		}
		distance := price.Sub(midPrice)
		if distance.LessThanOrEqual(maxDistance) {
			filteredAsks = append(filteredAsks, ask)
		}
	}

	return filteredBids, filteredAsks
}

// convertDepthUpdate converts Coinbase depth update to canonical format
func (e *SpotExchange) convertDepthUpdate(event *Event) *exchange.DepthUpdate {
	var bids []exchange.PriceLevel
	var asks []exchange.PriceLevel

	for _, update := range event.Updates {
		priceLevel := exchange.PriceLevel{
			Price:    update.PriceLevel,
			Quantity: update.NewQuantity,
		}

		if update.Side == "bid" {
			bids = append(bids, priceLevel)
		} else if update.Side == "ask" || update.Side == "offer" {
			asks = append(asks, priceLevel)
		}
	}

	eventTime := time.Now()

	return &exchange.DepthUpdate{
		Exchange:      e.GetName(),
		Symbol:        event.ProductID,
		EventTime:     eventTime,
		FirstUpdateID: 0,
		FinalUpdateID: 0,
		PrevUpdateID:  0,
		Bids:          bids,
		Asks:          asks,
		IsSnapshot:    event.Type == "snapshot",
	}
}

// convertToCoinbaseSymbol converts various symbol formats to Coinbase format
// Examples: BTCUSDT -> BTC-USD, BTC-USD -> BTC-USD
func convertToCoinbaseSymbol(symbol string) string {
	if strings.Contains(symbol, "-") {
		return strings.ToUpper(symbol)
	}

	symbol = strings.ToUpper(symbol)

	if strings.HasSuffix(symbol, "USDT") {
		base := strings.TrimSuffix(symbol, "USDT")
		return fmt.Sprintf("%s-USD", base)
	}

	if strings.HasSuffix(symbol, "USD") && !strings.HasSuffix(symbol, "USDT") {
		base := strings.TrimSuffix(symbol, "USD")
		return fmt.Sprintf("%s-USD", base)
	}

	if strings.HasSuffix(symbol, "USDC") {
		base := strings.TrimSuffix(symbol, "USDC")
		return fmt.Sprintf("%s-USDC", base)
	}

	log.Printf("[Coinbase] Warning: Could not convert symbol %s to Coinbase format, using as-is", symbol)
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
