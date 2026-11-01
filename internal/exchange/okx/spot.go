package okx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"orderbook/internal/exchange"
)

const (
	pollInterval = 1 * time.Second
	restBaseURL  = "https://www.okx.com/api/v5/market/books-full"
)

// SpotExchange implements the Exchange interface for OKX using REST polling
type SpotExchange struct {
	symbol     string
	instId     string // OKX format (e.g., BTC-USDT)
	restURL    string
	updateChan chan *exchange.DepthUpdate
	done       chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	health     atomic.Value
	isRunning  bool
}

// NewSpotExchange creates a new OKX Spot exchange instance
func NewSpotExchange(config Config) *SpotExchange {
	ctx, cancel := context.WithCancel(context.Background())

	instId := convertToOKXSymbol(config.Symbol)
	restURL := fmt.Sprintf("%s?instId=%s&sz=5000", restBaseURL, instId)

	ex := &SpotExchange{
		symbol:     config.Symbol,
		instId:     instId,
		restURL:    restURL,
		updateChan: make(chan *exchange.DepthUpdate, 1000),
		done:       make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
		isRunning:  false,
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
	return exchange.OKX
}

// GetSymbol returns the trading symbol
func (e *SpotExchange) GetSymbol() string {
	return e.symbol
}

// Connect starts the REST polling loop
func (e *SpotExchange) Connect(ctx context.Context) error {
	e.updateConnectionStatus(true)
	log.Printf("[%s] Starting REST polling (interval: %v)", e.GetName(), pollInterval)

	e.isRunning = true
	go e.pollLoop()

	return nil
}

// Close stops the polling loop
func (e *SpotExchange) Close() error {
	if e.cancel != nil {
		e.cancel()
	}

	select {
	case <-e.done:
	default:
		close(e.done)
	}

	e.updateConnectionStatus(false)
	log.Printf("[%s] Polling stopped", e.GetName())
	return nil
}

// GetSnapshot fetches the orderbook snapshot via REST API (5000 levels)
func (e *SpotExchange) GetSnapshot(ctx context.Context) (*exchange.Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", e.restURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		e.incrementErrorCount()
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	defer resp.Body.Close()

	var okxResp OrderBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&okxResp); err != nil {
		e.incrementErrorCount()
		return nil, fmt.Errorf("failed to decode snapshot: %w", err)
	}

	if okxResp.Code != "0" {
		e.incrementErrorCount()
		return nil, fmt.Errorf("API error: code=%s, msg=%s", okxResp.Code, okxResp.Msg)
	}

	if len(okxResp.Data) == 0 {
		e.incrementErrorCount()
		return nil, fmt.Errorf("empty response data")
	}

	snapshot := e.convertSnapshot(&okxResp.Data[0])
	return snapshot, nil
}

// Updates returns a channel that receives depth updates
func (e *SpotExchange) Updates() <-chan *exchange.DepthUpdate {
	return e.updateChan
}

// IsConnected checks if the polling is active
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

// pollLoop continuously polls REST endpoint every second
func (e *SpotExchange) pollLoop() {
	defer close(e.updateChan)
	defer e.updateConnectionStatus(false)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			log.Printf("[%s] Context cancelled, stopping polling", e.GetName())
			return
		case <-e.done:
			return
		case <-ticker.C:
			e.poll()
		}
	}
}

// poll fetches snapshot and sends as update
func (e *SpotExchange) poll() {
	ctx, cancel := context.WithTimeout(e.ctx, 5*time.Second)
	defer cancel()

	snapshot, err := e.GetSnapshot(ctx)
	if err != nil {
		log.Printf("[%s] Failed to poll: %v", e.GetName(), err)
		return
	}

	e.incrementMessageCount()
	e.updateLastPing()

	update := &exchange.DepthUpdate{
		Exchange:      e.GetName(),
		Symbol:        e.instId,
		EventTime:     snapshot.Timestamp,
		FirstUpdateID: 0,
		FinalUpdateID: 0,
		PrevUpdateID:  0,
		Bids:          snapshot.Bids,
		Asks:          snapshot.Asks,
		// books-full returns the entire book, so it replaces the local book.
		IsSnapshot: true,
	}

	select {
	case e.updateChan <- update:
	case <-e.ctx.Done():
	case <-e.done:
	}
}

// convertSnapshot converts OKX REST snapshot to canonical format
func (e *SpotExchange) convertSnapshot(data *OrderBookData) *exchange.Snapshot {
	bids := make([]exchange.PriceLevel, len(data.Bids))
	for i, bid := range data.Bids {
		if len(bid) >= 2 {
			bids[i] = exchange.PriceLevel{
				Price:    bid[0],
				Quantity: bid[1],
			}
		}
	}

	asks := make([]exchange.PriceLevel, len(data.Asks))
	for i, ask := range data.Asks {
		if len(ask) >= 2 {
			asks[i] = exchange.PriceLevel{
				Price:    ask[0],
				Quantity: ask[1],
			}
		}
	}

	return &exchange.Snapshot{
		Exchange:     e.GetName(),
		Symbol:       e.instId,
		LastUpdateID: 0,
		Bids:         bids,
		Asks:         asks,
		Timestamp:    time.Now(),
	}
}

// convertToOKXSymbol converts various symbol formats to OKX format
// Examples: BTCUSDT -> BTC-USDT, BTC-USDT -> BTC-USDT
func convertToOKXSymbol(symbol string) string {
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

	log.Printf("[OKX] Warning: Could not convert symbol %s to OKX format, using as-is", symbol)
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
