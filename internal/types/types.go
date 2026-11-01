package types

import (
	"time"

	"github.com/shopspring/decimal"
)

// TickLevel represents available tick size options for price aggregation
type TickLevel float64

const (
	Tick01  TickLevel = 0.1
	Tick1   TickLevel = 1.0
	Tick10  TickLevel = 10.0
	Tick50  TickLevel = 50.0
	Tick100 TickLevel = 100.0
)

// AvailableTickLevels defines the available tick levels in order of precision
var AvailableTickLevels = []TickLevel{
	Tick01,
	Tick1,
	Tick10,
	Tick50,
	Tick100,
}

// PriceLevel represents a single price level in the order book
type PriceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

// Stats holds statistical information about the order book
type Stats struct {
	EventsProcessed int64
	LastEventTime   time.Time
	ConnectionTime  time.Time
	BufferedEvents  int
	SequenceGaps    int64
	InvalidLevels   int64
	BidLevels       int
	AskLevels       int
	BestBid         decimal.Decimal
	BestAsk         decimal.Decimal
	Spread          decimal.Decimal

	// Liquidity depth metrics (in base asset units)
	BidLiquidity05Pct decimal.Decimal // Total bid size within 0.5% of mid
	AskLiquidity05Pct decimal.Decimal // Total ask size within 0.5% of mid
	BidLiquidity2Pct  decimal.Decimal // Total bid size within 2% of mid
	AskLiquidity2Pct  decimal.Decimal // Total ask size within 2% of mid
	BidLiquidity10Pct decimal.Decimal // Total bid size within 10% of mid
	AskLiquidity10Pct decimal.Decimal // Total ask size within 10% of mid

	// Liquidity imbalance (positive = more bids, negative = more asks)
	DeltaLiquidity05Pct decimal.Decimal // BidLiquidity05Pct - AskLiquidity05Pct
	DeltaLiquidity2Pct  decimal.Decimal // BidLiquidity2Pct - AskLiquidity2Pct
	DeltaLiquidity10Pct decimal.Decimal // BidLiquidity10Pct - AskLiquidity10Pct

	// Total quantities across all price levels
	TotalBidsQty decimal.Decimal // Sum of all bid quantities
	TotalAsksQty decimal.Decimal // Sum of all ask quantities
	TotalDelta   decimal.Decimal // TotalBidsQty - TotalAsksQty (positive = more bids)
}

// GetNextTickLevel returns the next tick level in the sequence
func GetNextTickLevel(current TickLevel) TickLevel {
	for i, tick := range AvailableTickLevels {
		if tick == current {
			// Return next tick level, or wrap around to first
			if i+1 < len(AvailableTickLevels) {
				return AvailableTickLevels[i+1]
			}
			return AvailableTickLevels[0]
		}
	}
	// If current not found, return first available
	return AvailableTickLevels[0]
}

// GetPreviousTickLevel returns the previous tick level in the sequence
func GetPreviousTickLevel(current TickLevel) TickLevel {
	for i, tick := range AvailableTickLevels {
		if tick == current {
			// Return previous tick level, or wrap around to last
			if i-1 >= 0 {
				return AvailableTickLevels[i-1]
			}
			return AvailableTickLevels[len(AvailableTickLevels)-1]
		}
	}
	// If current not found, return first available
	return AvailableTickLevels[0]
}
