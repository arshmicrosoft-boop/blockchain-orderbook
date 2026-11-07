package aggregation

import (
	"github.com/shopspring/decimal"
	"orderbook/internal/types"
)

// Aggregator handles price aggregation based on tick levels
type Aggregator struct {
	currentTick types.TickLevel
}

// New creates a new Aggregator instance
func New(tick types.TickLevel) *Aggregator {
	return &Aggregator{
		currentTick: tick,
	}
}

// SetTickLevel updates the tick level for aggregation
func (a *Aggregator) SetTickLevel(tick types.TickLevel) {
	a.currentTick = tick
}

// GetTickLevel returns the current tick level
func (a *Aggregator) GetTickLevel() types.TickLevel {
	return a.currentTick
}

// AggregateBids aggregates bid price levels by tick size (floors prices)
func (a *Aggregator) AggregateBids(levels []types.PriceLevel) []types.PriceLevel {
	if len(levels) == 0 {
		return levels
	}

	tickMap := make(map[string]types.PriceLevel)

	for _, level := range levels {
		roundedPrice := a.roundToTickBid(level.Price)
		key := roundedPrice.String()

		if existing, exists := tickMap[key]; exists {
			// Aggregate quantity
			tickMap[key] = types.PriceLevel{
				Price:    roundedPrice,
				Quantity: existing.Quantity.Add(level.Quantity),
			}
		} else {
			tickMap[key] = types.PriceLevel{
				Price:    roundedPrice,
				Quantity: level.Quantity,
			}
		}
	}

	// Convert map back to slice
	aggregated := make([]types.PriceLevel, 0, len(tickMap))
	for _, level := range tickMap {
		aggregated = append(aggregated, level)
	}

	return aggregated
}

// AggregateAsks aggregates ask price levels by tick size (ceils prices)
func (a *Aggregator) AggregateAsks(levels []types.PriceLevel) []types.PriceLevel {
	if len(levels) == 0 {
		return levels
	}

	tickMap := make(map[string]types.PriceLevel)

	for _, level := range levels {
		roundedPrice := a.roundToTickAsk(level.Price)
		key := roundedPrice.String()

		if existing, exists := tickMap[key]; exists {
			// Aggregate quantity
			tickMap[key] = types.PriceLevel{
				Price:    roundedPrice,
				Quantity: existing.Quantity.Add(level.Quantity),
			}
		} else {
			tickMap[key] = types.PriceLevel{
				Price:    roundedPrice,
				Quantity: level.Quantity,
			}
		}
	}

	// Convert map back to slice
	aggregated := make([]types.PriceLevel, 0, len(tickMap))
	for _, level := range tickMap {
		aggregated = append(aggregated, level)
	}

	return aggregated
}

// roundToTickBid rounds a bid price DOWN to maintain proper spread
func (a *Aggregator) roundToTickBid(price decimal.Decimal) decimal.Decimal {
	tickSize := decimal.NewFromFloat(float64(a.currentTick))
	if tickSize.IsZero() {
		return price
	}

	// Floor bids: floor(price / tickSize) * tickSize
	divided := price.Div(tickSize)
	floored := divided.Floor() // Floor to lower integer
	return floored.Mul(tickSize)
}

// roundToTickAsk rounds an ask price UP to maintain proper spread
func (a *Aggregator) roundToTickAsk(price decimal.Decimal) decimal.Decimal {
	tickSize := decimal.NewFromFloat(float64(a.currentTick))
	if tickSize.IsZero() {
		return price
	}

	// Ceiling asks: ceil(price / tickSize) * tickSize
	divided := price.Div(tickSize)
	ceiled := divided.Ceil() // Ceiling to higher integer
	return ceiled.Mul(tickSize)
}

// FilterLevels filters price levels based on best ask price to remove outliers
func FilterLevels(levels []types.PriceLevel, bestAsk decimal.Decimal, isBid bool) []types.PriceLevel {
	if bestAsk.IsZero() {
		return levels
	}

	filtered := make([]types.PriceLevel, 0, len(levels))
	maxPrice := bestAsk.Mul(decimal.NewFromFloat(2.0))
	minPrice := bestAsk.Mul(decimal.NewFromFloat(0.2))

	for _, level := range levels {
		if isBid {
			// For bids, filter out prices that are too high or too low
			if level.Price.LessThanOrEqual(maxPrice) && level.Price.GreaterThanOrEqual(minPrice) {
				filtered = append(filtered, level)
			}
		} else {
			// For asks, no additional filtering needed beyond basic sanity checks
			filtered = append(filtered, level)
		}
	}

	return filtered
}
