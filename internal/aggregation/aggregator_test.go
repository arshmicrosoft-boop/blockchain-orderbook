package aggregation

import (
	"testing"

	"github.com/shopspring/decimal"
	"orderbook/internal/types"
)

func TestNew(t *testing.T) {
	tick := types.Tick1
	agg := New(tick)

	if agg == nil {
		t.Fatal("New() returned nil")
	}

	if agg.GetTickLevel() != tick {
		t.Errorf("Expected tick level %g, got %g", float64(tick), float64(agg.GetTickLevel()))
	}
}

func TestSetGetTickLevel(t *testing.T) {
	agg := New(types.Tick1)

	newTick := types.Tick10
	agg.SetTickLevel(newTick)

	if agg.GetTickLevel() != newTick {
		t.Errorf("Expected tick level %g, got %g", float64(newTick), float64(agg.GetTickLevel()))
	}
}

func TestAggregateBids(t *testing.T) {
	tests := []struct {
		name     string
		tick     types.TickLevel
		levels   []types.PriceLevel
		expected int
	}{
		{
			name: "No aggregation needed - tick 0.1",
			tick: types.Tick01,
			levels: []types.PriceLevel{
				{Price: decimal.NewFromFloat(50000.1), Quantity: decimal.NewFromFloat(1.0)},
				{Price: decimal.NewFromFloat(50000.2), Quantity: decimal.NewFromFloat(1.5)},
			},
			expected: 2,
		},
		{
			name: "Aggregation needed - tick 1.0",
			tick: types.Tick1,
			levels: []types.PriceLevel{
				{Price: decimal.NewFromFloat(50000.1), Quantity: decimal.NewFromFloat(1.0)},
				{Price: decimal.NewFromFloat(50000.9), Quantity: decimal.NewFromFloat(1.5)},
			},
			expected: 1, // Both should aggregate to 50000.0
		},
		{
			name: "Aggregation needed - tick 10.0",
			tick: types.Tick10,
			levels: []types.PriceLevel{
				{Price: decimal.NewFromFloat(50001), Quantity: decimal.NewFromFloat(1.0)},
				{Price: decimal.NewFromFloat(50005), Quantity: decimal.NewFromFloat(1.5)},
				{Price: decimal.NewFromFloat(50009), Quantity: decimal.NewFromFloat(2.0)},
			},
			expected: 1, // All should aggregate to 50000.0
		},
		{
			name:     "Empty levels",
			tick:     types.Tick1,
			levels:   []types.PriceLevel{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := New(tt.tick)
			result := agg.AggregateBids(tt.levels)

			if len(result) != tt.expected {
				t.Errorf("Expected %d aggregated levels, got %d", tt.expected, len(result))
			}

			// Check that quantities are properly aggregated
			if len(result) == 1 && len(tt.levels) > 1 {
				expectedQty := decimal.Zero
				for _, level := range tt.levels {
					expectedQty = expectedQty.Add(level.Quantity)
				}

				if !result[0].Quantity.Equal(expectedQty) {
					t.Errorf("Expected aggregated quantity %s, got %s",
						expectedQty.String(), result[0].Quantity.String())
				}
			}
		})
	}
}

func TestAggregateAsks(t *testing.T) {
	tests := []struct {
		name     string
		tick     types.TickLevel
		levels   []types.PriceLevel
		expected int
	}{
		{
			name: "No aggregation needed - tick 0.1",
			tick: types.Tick01,
			levels: []types.PriceLevel{
				{Price: decimal.NewFromFloat(50001.1), Quantity: decimal.NewFromFloat(1.0)},
				{Price: decimal.NewFromFloat(50001.2), Quantity: decimal.NewFromFloat(1.5)},
			},
			expected: 2,
		},
		{
			name: "Aggregation needed - tick 1.0",
			tick: types.Tick1,
			levels: []types.PriceLevel{
				{Price: decimal.NewFromFloat(50001.1), Quantity: decimal.NewFromFloat(1.0)},
				{Price: decimal.NewFromFloat(50001.9), Quantity: decimal.NewFromFloat(1.5)},
			},
			expected: 1, // Both should aggregate to 50002.0 (ceiling)
		},
		{
			name: "Aggregation needed - tick 10.0",
			tick: types.Tick10,
			levels: []types.PriceLevel{
				{Price: decimal.NewFromFloat(50001), Quantity: decimal.NewFromFloat(1.0)},
				{Price: decimal.NewFromFloat(50005), Quantity: decimal.NewFromFloat(1.5)},
				{Price: decimal.NewFromFloat(50009), Quantity: decimal.NewFromFloat(2.0)},
			},
			expected: 1, // All should aggregate to 50010.0 (ceiling)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := New(tt.tick)
			result := agg.AggregateAsks(tt.levels)

			if len(result) != tt.expected {
				t.Errorf("Expected %d aggregated levels, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestRoundToTickBid(t *testing.T) {
	tests := []struct {
		name     string
		tick     types.TickLevel
		price    decimal.Decimal
		expected decimal.Decimal
	}{
		{
			name:     "Round down tick 1.0",
			tick:     types.Tick1,
			price:    decimal.NewFromFloat(50000.9),
			expected: decimal.NewFromFloat(50000.0),
		},
		{
			name:     "Round down tick 10.0",
			tick:     types.Tick10,
			price:    decimal.NewFromFloat(50005),
			expected: decimal.NewFromFloat(50000),
		},
		{
			name:     "Already aligned",
			tick:     types.Tick1,
			price:    decimal.NewFromFloat(50000.0),
			expected: decimal.NewFromFloat(50000.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := New(tt.tick)
			result := agg.roundToTickBid(tt.price)

			if !result.Equal(tt.expected) {
				t.Errorf("Expected %s, got %s", tt.expected.String(), result.String())
			}
		})
	}
}

func TestRoundToTickAsk(t *testing.T) {
	tests := []struct {
		name     string
		tick     types.TickLevel
		price    decimal.Decimal
		expected decimal.Decimal
	}{
		{
			name:     "Round up tick 1.0",
			tick:     types.Tick1,
			price:    decimal.NewFromFloat(50000.1),
			expected: decimal.NewFromFloat(50001.0),
		},
		{
			name:     "Round up tick 10.0",
			tick:     types.Tick10,
			price:    decimal.NewFromFloat(50001),
			expected: decimal.NewFromFloat(50010),
		},
		{
			name:     "Already aligned",
			tick:     types.Tick1,
			price:    decimal.NewFromFloat(50000.0),
			expected: decimal.NewFromFloat(50000.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := New(tt.tick)
			result := agg.roundToTickAsk(tt.price)

			if !result.Equal(tt.expected) {
				t.Errorf("Expected %s, got %s", tt.expected.String(), result.String())
			}
		})
	}
}

func TestFilterLevels(t *testing.T) {
	bestAsk := decimal.NewFromFloat(50000)

	levels := []types.PriceLevel{
		{Price: decimal.NewFromFloat(49000), Quantity: decimal.NewFromFloat(1.0)},  // Valid
		{Price: decimal.NewFromFloat(45000), Quantity: decimal.NewFromFloat(1.0)},  // Valid
		{Price: decimal.NewFromFloat(5000), Quantity: decimal.NewFromFloat(1.0)},   // Too low
		{Price: decimal.NewFromFloat(150000), Quantity: decimal.NewFromFloat(1.0)}, // Too high
	}

	filtered := FilterLevels(levels, bestAsk, true)

	expectedCount := 2
	if len(filtered) != expectedCount {
		t.Errorf("Expected %d filtered levels, got %d", expectedCount, len(filtered))
	}
}

// Benchmarks

func BenchmarkAggregateBids(b *testing.B) {
	agg := New(types.Tick1)

	// Create test data
	levels := make([]types.PriceLevel, 1000)
	for i := 0; i < 1000; i++ {
		levels[i] = types.PriceLevel{
			Price:    decimal.NewFromFloat(50000 - float64(i) + 0.5), // Add 0.5 to force aggregation
			Quantity: decimal.NewFromFloat(1.0),
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		agg.AggregateBids(levels)
	}
}

func BenchmarkAggregateAsks(b *testing.B) {
	agg := New(types.Tick1)

	// Create test data
	levels := make([]types.PriceLevel, 1000)
	for i := 0; i < 1000; i++ {
		levels[i] = types.PriceLevel{
			Price:    decimal.NewFromFloat(50001 + float64(i) + 0.5), // Add 0.5 to force aggregation
			Quantity: decimal.NewFromFloat(1.0),
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		agg.AggregateAsks(levels)
	}
}

func BenchmarkFilterLevels(b *testing.B) {
	bestAsk := decimal.NewFromFloat(50000)

	// Create test data
	levels := make([]types.PriceLevel, 5000)
	for i := 0; i < 5000; i++ {
		levels[i] = types.PriceLevel{
			Price:    decimal.NewFromFloat(float64(i * 10)), // Wide range
			Quantity: decimal.NewFromFloat(1.0),
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		FilterLevels(levels, bestAsk, true)
	}
}
