package orderbook

import (
	"log"
	"sync"
	"time"

	"orderbook/internal/exchange"
	"orderbook/internal/types"

	"github.com/shopspring/decimal"
)

// maxBufferedEvents is the number of out-of-sequence updates we tolerate before
// asking for a fresh snapshot.
const maxBufferedEvents = 100

// OrderBook manages the real-time order book state.
type OrderBook struct {
	mu           sync.RWMutex
	bids         map[string]types.PriceLevel
	asks         map[string]types.PriceLevel
	lastUpdateID int64
	eventBuffer  []*exchange.DepthUpdate
	initialized  bool
	needsResync  bool
	stats        types.Stats
	currentTick  types.TickLevel

	// Cached best bid/ask for performance
	bestBid   decimal.Decimal
	bestAsk   decimal.Decimal
	bidLevels int
	askLevels int

	// Liquidity metrics are O(n) over the whole book. They are recomputed only
	// when the book changed and somebody actually asks for stats (the server
	// does that a few times per second), not on every depth update.
	liquidityDirty bool
}

// New creates a new OrderBook instance.
func New() *OrderBook {
	return &OrderBook{
		bids:        make(map[string]types.PriceLevel),
		asks:        make(map[string]types.PriceLevel),
		eventBuffer: make([]*exchange.DepthUpdate, 0),
		currentTick: types.Tick1, // Default to 1.0 tick size
		stats: types.Stats{
			ConnectionTime: time.Now(),
		},
	}
}

// LoadSnapshot initializes the orderbook with a full snapshot from the exchange.
//
// Invalid price levels are skipped and counted instead of aborting the whole
// snapshot: previously a single malformed level left the book in a partially
// loaded state and returned an error.
func (ob *OrderBook) LoadSnapshot(snapshot *exchange.Snapshot) error {
	if snapshot == nil {
		return nil
	}

	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.bids = make(map[string]types.PriceLevel, len(snapshot.Bids))
	ob.asks = make(map[string]types.PriceLevel, len(snapshot.Asks))
	ob.lastUpdateID = snapshot.LastUpdateID
	ob.needsResync = false

	// Reset cached top-of-book. Without this a resync to a book with lower
	// prices kept the previous best bid/ask forever.
	ob.bestBid = decimal.Zero
	ob.bestAsk = decimal.Zero

	for _, bid := range snapshot.Bids {
		ob.setLevelLocked(ob.bids, bid, true)
	}
	for _, ask := range snapshot.Asks {
		ob.setLevelLocked(ob.asks, ask, false)
	}

	ob.recalculateBestBidLocked()
	ob.recalculateBestAskLocked()
	ob.updateCachedStatsLocked()

	return nil
}

// HandleDepthUpdate processes a depth update from the exchange stream.
func (ob *OrderBook) HandleDepthUpdate(update *exchange.DepthUpdate) {
	if update == nil {
		return
	}

	ob.mu.Lock()
	defer ob.mu.Unlock()

	if !ob.initialized {
		ob.bufferEventLocked(update)
		return
	}

	// Full-book messages (Hyperliquid, OKX polling, periodic exchange
	// snapshots) are authoritative replacements, not deltas.
	if update.IsSnapshot || isUnsequenced(update) {
		ob.applyUpdateLocked(update)
		return
	}

	// Updates already covered by the snapshot (or a previous update) are
	// stale. This happens routinely for events that were in flight while the
	// REST snapshot was being fetched; treating them as a gap caused endless
	// reconnect/resync loops.
	if update.FinalUpdateID <= ob.lastUpdateID {
		return
	}

	expectedPrevID := ob.lastUpdateID
	if update.PrevUpdateID == expectedPrevID {
		ob.applyUpdateLocked(update)
		return
	}

	// Binance-style overlapping event: U <= lastUpdateId+1 <= u.
	if update.FirstUpdateID <= expectedPrevID+1 && update.FinalUpdateID > expectedPrevID {
		ob.applyUpdateLocked(update)
		return
	}

	// A real gap. Buffer the event and ask for a fresh snapshot; applying it
	// would silently corrupt the book.
	if !ob.needsResync {
		ob.stats.SequenceGaps++
		log.Printf("Sequence gap: lastUpdateId=%d, event U=%d u=%d pu=%d",
			expectedPrevID, update.FirstUpdateID, update.FinalUpdateID, update.PrevUpdateID)
	}
	ob.needsResync = true
	ob.bufferEventLocked(update)
}

// ProcessBufferedEvents applies events that arrived while the snapshot was
// being loaded. Unsequenced events (no update IDs) are applied in arrival
// order; sequenced events are applied only while they are contiguous.
func (ob *OrderBook) ProcessBufferedEvents() {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	events := ob.eventBuffer
	ob.eventBuffer = nil

	applied := 0
	for i, event := range events {
		if event == nil {
			continue
		}

		if event.IsSnapshot || isUnsequenced(event) {
			ob.applyUpdateLocked(event)
			applied++
			continue
		}

		if event.FinalUpdateID <= ob.lastUpdateID {
			// Already covered by the snapshot.
			continue
		}

		if event.FirstUpdateID <= ob.lastUpdateID+1 {
			ob.applyUpdateLocked(event)
			applied++
			continue
		}

		// Genuine gap: everything after it is unusable until we resync.
		ob.stats.SequenceGaps++
		ob.needsResync = true
		log.Printf("Sequence gap after %d buffered events (U=%d, u=%d, lastUpdateId=%d); scheduling resync",
			i, event.FirstUpdateID, event.FinalUpdateID, ob.lastUpdateID)
		break
	}

	ob.initialized = true
	ob.updateCachedStatsLocked()
	log.Printf("Orderbook initialized with %d valid buffered events", applied)
}

// NeedsResync reports whether the book detected a sequence gap or buffered more
// events than it is willing to hold. The supervisor reacts by reconnecting so
// the adapter can provide a fresh snapshot.
func (ob *OrderBook) NeedsResync() bool {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.needsResync || len(ob.eventBuffer) > maxBufferedEvents
}

// SetTickLevel changes the current tick level for price aggregation.
func (ob *OrderBook) SetTickLevel(tick types.TickLevel) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	ob.currentTick = tick
}

// GetTickLevel returns the current tick level.
func (ob *OrderBook) GetTickLevel() types.TickLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.currentTick
}

// SnapshotLevels returns copies of the current bid and ask levels. It is
// cheaper than calling GetBids and GetAsks separately because it takes the
// lock once and builds slices instead of intermediate maps.
func (ob *OrderBook) SnapshotLevels() ([]types.PriceLevel, []types.PriceLevel) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := make([]types.PriceLevel, 0, len(ob.bids))
	for _, level := range ob.bids {
		bids = append(bids, level)
	}

	asks := make([]types.PriceLevel, 0, len(ob.asks))
	for _, level := range ob.asks {
		asks = append(asks, level)
	}

	return bids, asks
}

// GetBids returns a copy of the current bid levels.
func (ob *OrderBook) GetBids() map[string]types.PriceLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := make(map[string]types.PriceLevel, len(ob.bids))
	for k, v := range ob.bids {
		bids[k] = v
	}
	return bids
}

// GetAsks returns a copy of the current ask levels.
func (ob *OrderBook) GetAsks() map[string]types.PriceLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	asks := make(map[string]types.PriceLevel, len(ob.asks))
	for k, v := range ob.asks {
		asks[k] = v
	}
	return asks
}

// GetStats returns a consistent copy of the current statistics. Liquidity
// metrics are refreshed lazily, at most once per book change.
func (ob *OrderBook) GetStats() types.Stats {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if ob.liquidityDirty {
		ob.calculateLiquidityDepthLocked()
		ob.liquidityDirty = false
	}

	return ob.stats
}

// IsInitialized returns whether the orderbook is initialized.
func (ob *OrderBook) IsInitialized() bool {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.initialized
}

// GetBufferLength returns the current buffer length.
func (ob *OrderBook) GetBufferLength() int {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return len(ob.eventBuffer)
}

// isUnsequenced reports whether the adapter provides no update identifiers
// (Kraken, Coinbase). Such updates are applied in arrival order.
func isUnsequenced(update *exchange.DepthUpdate) bool {
	return update.FirstUpdateID == 0 && update.FinalUpdateID == 0 && update.PrevUpdateID == 0
}

// bufferEventLocked appends an event to the pending buffer. The buffer is
// capped so a slow snapshot fetch cannot grow memory without bound; once the
// cap is hit the book is marked for resync and further events are dropped.
func (ob *OrderBook) bufferEventLocked(update *exchange.DepthUpdate) {
	if len(ob.eventBuffer) >= maxBufferedEvents {
		ob.needsResync = true
		ob.stats.BufferedEvents = len(ob.eventBuffer)
		return
	}
	ob.eventBuffer = append(ob.eventBuffer, update)
	ob.stats.BufferedEvents = len(ob.eventBuffer)
}

// applyUpdateLocked applies one depth update (must be called with the mutex
// held).
func (ob *OrderBook) applyUpdateLocked(update *exchange.DepthUpdate) {
	if update.IsSnapshot {
		ob.bids = make(map[string]types.PriceLevel, len(update.Bids))
		ob.asks = make(map[string]types.PriceLevel, len(update.Asks))
		ob.bestBid = decimal.Zero
		ob.bestAsk = decimal.Zero
	}

	bestBidRemoved := false
	bestAskRemoved := false

	for _, bid := range update.Bids {
		price, qty, ok := parseLevel(bid)
		if !ok {
			ob.stats.InvalidLevels++
			continue
		}

		// Canonical decimal key: exchanges disagree on formatting ("100.10"
		// vs "100.1"), and a mismatched key made deletions silently no-op.
		key := price.String()

		if qty.IsZero() {
			if _, exists := ob.bids[key]; exists {
				delete(ob.bids, key)
				if price.Equal(ob.bestBid) {
					bestBidRemoved = true
				}
			}
			continue
		}

		ob.bids[key] = types.PriceLevel{Price: price, Quantity: qty}
		if ob.bestBid.IsZero() || price.GreaterThan(ob.bestBid) {
			ob.bestBid = price
		}
	}

	for _, ask := range update.Asks {
		price, qty, ok := parseLevel(ask)
		if !ok {
			ob.stats.InvalidLevels++
			continue
		}

		key := price.String()

		if qty.IsZero() {
			if _, exists := ob.asks[key]; exists {
				delete(ob.asks, key)
				if price.Equal(ob.bestAsk) {
					bestAskRemoved = true
				}
			}
			continue
		}

		ob.asks[key] = types.PriceLevel{Price: price, Quantity: qty}
		if ob.bestAsk.IsZero() || price.LessThan(ob.bestAsk) {
			ob.bestAsk = price
		}
	}

	if update.IsSnapshot {
		// Recompute from scratch: messages may omit the current best level.
		ob.recalculateBestBidLocked()
		ob.recalculateBestAskLocked()
	} else {
		if bestBidRemoved {
			ob.recalculateBestBidLocked()
		}
		if bestAskRemoved {
			ob.recalculateBestAskLocked()
		}
	}

	ob.lastUpdateID = update.FinalUpdateID
	ob.stats.EventsProcessed++
	ob.stats.LastEventTime = update.EventTime
	ob.updateCachedStatsLocked()
}

// setLevelLocked inserts a level, ignoring malformed or empty levels. Used
// while loading a snapshot.
func (ob *OrderBook) setLevelLocked(side map[string]types.PriceLevel, level exchange.PriceLevel, isBid bool) {
	price, qty, ok := parseLevel(level)
	if !ok {
		ob.stats.InvalidLevels++
		return
	}
	if qty.IsZero() {
		return
	}

	side[price.String()] = types.PriceLevel{Price: price, Quantity: qty}
	if isBid {
		if ob.bestBid.IsZero() || price.GreaterThan(ob.bestBid) {
			ob.bestBid = price
		}
	} else if ob.bestAsk.IsZero() || price.LessThan(ob.bestAsk) {
		ob.bestAsk = price
	}
}

// parseLevel converts a wire level to decimals. A malformed level is skipped
// by the caller; it must never be treated as a zero-quantity deletion, which
// is what the old implementation did after ignoring the parse error.
func parseLevel(level exchange.PriceLevel) (decimal.Decimal, decimal.Decimal, bool) {
	price, err := decimal.NewFromString(level.Price)
	if err != nil || price.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, decimal.Zero, false
	}

	qty, err := decimal.NewFromString(level.Quantity)
	if err != nil || qty.IsNegative() {
		return decimal.Zero, decimal.Zero, false
	}

	return price, qty, true
}

// updateCachedStatsLocked refreshes the cheap statistics fields (must be
// called with the mutex held).
func (ob *OrderBook) updateCachedStatsLocked() {
	ob.bidLevels = len(ob.bids)
	ob.askLevels = len(ob.asks)

	ob.stats.BidLevels = ob.bidLevels
	ob.stats.AskLevels = ob.askLevels
	ob.stats.BufferedEvents = len(ob.eventBuffer)
	ob.stats.BestBid = ob.bestBid
	ob.stats.BestAsk = ob.bestAsk

	if !ob.bestBid.IsZero() && !ob.bestAsk.IsZero() && ob.bestAsk.GreaterThan(ob.bestBid) {
		ob.stats.Spread = ob.bestAsk.Sub(ob.bestBid)
	} else {
		ob.stats.Spread = decimal.Zero
	}

	ob.liquidityDirty = true
}

// calculateLiquidityDepthLocked calculates liquidity at various depth
// percentages (must be called with the mutex held).
func (ob *OrderBook) calculateLiquidityDepthLocked() {
	if ob.bestBid.IsZero() || ob.bestAsk.IsZero() {
		ob.stats.BidLiquidity05Pct = decimal.Zero
		ob.stats.AskLiquidity05Pct = decimal.Zero
		ob.stats.BidLiquidity2Pct = decimal.Zero
		ob.stats.AskLiquidity2Pct = decimal.Zero
		ob.stats.BidLiquidity10Pct = decimal.Zero
		ob.stats.AskLiquidity10Pct = decimal.Zero
		ob.stats.DeltaLiquidity05Pct = decimal.Zero
		ob.stats.DeltaLiquidity2Pct = decimal.Zero
		ob.stats.DeltaLiquidity10Pct = decimal.Zero
		ob.stats.TotalBidsQty = decimal.Zero
		ob.stats.TotalAsksQty = decimal.Zero
		ob.stats.TotalDelta = decimal.Zero
		return
	}

	midPrice := ob.bestBid.Add(ob.bestAsk).Div(decimal.NewFromInt(2))

	threshold05Pct := midPrice.Mul(decimal.NewFromFloat(0.005))
	threshold2Pct := midPrice.Mul(decimal.NewFromFloat(0.02))
	threshold10Pct := midPrice.Mul(decimal.NewFromFloat(0.10))

	bidLiq05 := decimal.Zero
	bidLiq2 := decimal.Zero
	bidLiq10 := decimal.Zero
	totalBidsQty := decimal.Zero
	minBid05Pct := midPrice.Sub(threshold05Pct)
	minBid2Pct := midPrice.Sub(threshold2Pct)
	minBid10Pct := midPrice.Sub(threshold10Pct)

	for _, level := range ob.bids {
		totalBidsQty = totalBidsQty.Add(level.Quantity)
		if level.Price.GreaterThanOrEqual(minBid05Pct) {
			bidLiq05 = bidLiq05.Add(level.Quantity)
		}
		if level.Price.GreaterThanOrEqual(minBid2Pct) {
			bidLiq2 = bidLiq2.Add(level.Quantity)
		}
		if level.Price.GreaterThanOrEqual(minBid10Pct) {
			bidLiq10 = bidLiq10.Add(level.Quantity)
		}
	}

	askLiq05 := decimal.Zero
	askLiq2 := decimal.Zero
	askLiq10 := decimal.Zero
	totalAsksQty := decimal.Zero
	maxAsk05Pct := midPrice.Add(threshold05Pct)
	maxAsk2Pct := midPrice.Add(threshold2Pct)
	maxAsk10Pct := midPrice.Add(threshold10Pct)

	for _, level := range ob.asks {
		totalAsksQty = totalAsksQty.Add(level.Quantity)
		if level.Price.LessThanOrEqual(maxAsk05Pct) {
			askLiq05 = askLiq05.Add(level.Quantity)
		}
		if level.Price.LessThanOrEqual(maxAsk2Pct) {
			askLiq2 = askLiq2.Add(level.Quantity)
		}
		if level.Price.LessThanOrEqual(maxAsk10Pct) {
			askLiq10 = askLiq10.Add(level.Quantity)
		}
	}

	ob.stats.BidLiquidity05Pct = bidLiq05
	ob.stats.AskLiquidity05Pct = askLiq05
	ob.stats.BidLiquidity2Pct = bidLiq2
	ob.stats.AskLiquidity2Pct = askLiq2
	ob.stats.BidLiquidity10Pct = bidLiq10
	ob.stats.AskLiquidity10Pct = askLiq10
	ob.stats.TotalBidsQty = totalBidsQty
	ob.stats.TotalAsksQty = totalAsksQty

	// Positive delta = more bid liquidity = bullish pressure.
	ob.stats.DeltaLiquidity05Pct = bidLiq05.Sub(askLiq05)
	ob.stats.DeltaLiquidity2Pct = bidLiq2.Sub(askLiq2)
	ob.stats.DeltaLiquidity10Pct = bidLiq10.Sub(askLiq10)
	ob.stats.TotalDelta = totalBidsQty.Sub(totalAsksQty)
}

// recalculateBestBidLocked recalculates the best bid when the current best is
// removed.
func (ob *OrderBook) recalculateBestBidLocked() {
	ob.bestBid = decimal.Zero
	for _, level := range ob.bids {
		if ob.bestBid.IsZero() || level.Price.GreaterThan(ob.bestBid) {
			ob.bestBid = level.Price
		}
	}
}

// recalculateBestAskLocked recalculates the best ask when the current best is
// removed.
func (ob *OrderBook) recalculateBestAskLocked() {
	ob.bestAsk = decimal.Zero
	for _, level := range ob.asks {
		if ob.bestAsk.IsZero() || level.Price.LessThan(ob.bestAsk) {
			ob.bestAsk = level.Price
		}
	}
}
