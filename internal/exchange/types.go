package exchange

import (
	"context"
	"time"
)

// ExchangeName represents supported exchange identifiers
type ExchangeName string

const (
	Binancef     ExchangeName = "binancef"
	Binance      ExchangeName = "binance"
	Bybitf       ExchangeName = "bybitf"
	Bybit        ExchangeName = "bybit"
	Kraken       ExchangeName = "kraken"
	Hyperliquidf ExchangeName = "hyperliquidf"
	OKX          ExchangeName = "okx"
	Coinbase     ExchangeName = "coinbase"
	Asterdexf    ExchangeName = "asterdexf"
	BingX        ExchangeName = "bingx"
	BingXf       ExchangeName = "bingxf"
)

// Exchange defines the interface that all exchange adapters must implement
type Exchange interface {
	// GetName returns the exchange name (e.g., "binancef", "binance")
	GetName() ExchangeName

	// GetSymbol returns the trading symbol
	GetSymbol() string

	// Connect establishes connection to the exchange
	Connect(ctx context.Context) error

	// Close closes the connection gracefully
	Close() error

	// GetSnapshot fetches the initial orderbook snapshot
	GetSnapshot(ctx context.Context) (*Snapshot, error)

	// Updates returns a channel that receives depth updates in canonical format
	Updates() <-chan *DepthUpdate

	// IsConnected returns connection status
	IsConnected() bool

	// Health returns connection health information
	Health() HealthStatus
}

// Snapshot represents a canonical orderbook snapshot (normalized across exchanges)
type Snapshot struct {
	Exchange     ExchangeName // Exchange name
	Symbol       string       // Trading symbol
	LastUpdateID int64        // Last update ID from exchange
	Bids         []PriceLevel // Bid levels [price, quantity]
	Asks         []PriceLevel // Ask levels [price, quantity]
	Timestamp    time.Time    // Snapshot timestamp
}

// DepthUpdate represents a canonical depth update event (normalized across exchanges)
type DepthUpdate struct {
	Exchange      ExchangeName // Exchange name
	Symbol        string       // Trading symbol
	EventTime     time.Time    // Event timestamp
	FirstUpdateID int64        // First update ID in this event
	FinalUpdateID int64        // Final update ID in this event
	PrevUpdateID  int64        // Previous update ID (for continuity checking)
	Bids          []PriceLevel // Updated bid levels
	Asks          []PriceLevel // Updated ask levels
	// IsSnapshot marks a full-book message that must replace the current book
	// instead of being merged into it. Some adapters (Hyperliquid, OKX REST
	// polling, exchange resnapshots) only ever send complete books.
	IsSnapshot bool
}

// PriceLevel represents a single price level [price, quantity]
type PriceLevel struct {
	Price    string // Price as string to avoid precision loss
	Quantity string // Quantity as string to avoid precision loss
}

// HealthStatus represents connection health information
type HealthStatus struct {
	Connected     bool
	LastPing      time.Time
	MessageCount  int64
	ErrorCount    int64
	ReconnectTime *time.Time
}
