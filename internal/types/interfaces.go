package types

import (
	"time"
)

// PriceAggregator defines the interface for price aggregation
type PriceAggregator interface {
	// SetTickLevel updates the tick level for aggregation
	SetTickLevel(tick TickLevel)

	// GetTickLevel returns the current tick level
	GetTickLevel() TickLevel

	// AggregateBids aggregates bid price levels
	AggregateBids(levels []PriceLevel) []PriceLevel

	// AggregateAsks aggregates ask price levels
	AggregateAsks(levels []PriceLevel) []PriceLevel
}

// Display defines the interface for orderbook visualization
type Display interface {
	// DisplayOrderBook renders the orderbook
	DisplayOrderBook(bids, asks map[string]PriceLevel, stats Stats, initialized bool, bufferLen int)

	// SetAggregator updates the aggregator used for display
	SetAggregator(aggregator PriceAggregator)

	// Run starts the display (for interactive displays like Bubble Tea)
	Run() error

	// UpdateData updates the display with new data (for interactive displays)
	UpdateData(bids, asks map[string]PriceLevel, stats Stats, initialized bool, bufferLen int)

	// Quit signals the display to quit (for interactive displays)
	Quit()
}

// ConfigProvider defines the interface for configuration management
// Note: Implementations should import their own config types
type ConfigProvider interface {
	// GetSymbol returns the trading symbol
	GetSymbol() string

	// GetWebSocketURL returns the WebSocket URL
	GetWebSocketURL() string

	// GetRestURL returns the REST API URL
	GetRestURL() string

	// GetDisplayTop returns the number of levels to display
	GetDisplayTop() int

	// GetUpdateInterval returns the display update interval
	GetUpdateInterval() time.Duration

	// GetDefaultTickLevel returns the default tick level
	GetDefaultTickLevel() TickLevel
}
