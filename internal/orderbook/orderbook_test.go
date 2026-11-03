package orderbook

import (
	"testing"

	"orderbook/internal/exchange"
)

func wireLevel(price, qty string) exchange.PriceLevel {
	return exchange.PriceLevel{Price: price, Quantity: qty}
}

func snapshot(lastID int64, bids, asks []exchange.PriceLevel) *exchange.Snapshot {
	return &exchange.Snapshot{
		LastUpdateID: lastID,
		Bids:         bids,
		Asks:         asks,
	}
}

func newInitializedBook(t *testing.T, snap *exchange.Snapshot) *OrderBook {
	t.Helper()
	ob := New()
	if err := ob.LoadSnapshot(snap); err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	ob.ProcessBufferedEvents()
	return ob
}

func TestLoadSnapshotComputesBestPrices(t *testing.T) {
	ob := newInitializedBook(t, snapshot(42,
		[]exchange.PriceLevel{wireLevel("100.10", "2"), wireLevel("100.00", "1")},
		[]exchange.PriceLevel{wireLevel("100.20", "3"), wireLevel("100.30", "4")},
	))

	stats := ob.GetStats()
	if got, want := stats.BestBid.String(), "100.1"; got != want {
		t.Errorf("BestBid = %s, want %s", got, want)
	}
	if got, want := stats.BestAsk.String(), "100.2"; got != want {
		t.Errorf("BestAsk = %s, want %s", got, want)
	}
	if stats.BidLevels != 2 || stats.AskLevels != 2 {
		t.Errorf("levels = %d/%d, want 2/2", stats.BidLevels, stats.AskLevels)
	}
	if got, want := stats.TotalBidsQty.String(), "3"; got != want {
		t.Errorf("TotalBidsQty = %s, want %s", got, want)
	}
}

func TestDeltaRemovalRecalculatesBestBid(t *testing.T) {
	ob := newInitializedBook(t, snapshot(10,
		[]exchange.PriceLevel{wireLevel("100", "1"), wireLevel("99", "1")},
		[]exchange.PriceLevel{wireLevel("101", "1")},
	))

	ob.HandleDepthUpdate(&exchange.DepthUpdate{
		FinalUpdateID: 11,
		PrevUpdateID:  10,
		Bids:          []exchange.PriceLevel{wireLevel("100", "0")},
	})

	stats := ob.GetStats()
	if got, want := stats.BestBid.String(), "99"; got != want {
		t.Errorf("BestBid after removal = %s, want %s", got, want)
	}
	if stats.BidLevels != 1 {
		t.Errorf("BidLevels = %d, want 1", stats.BidLevels)
	}
}

// A malformed level used to parse as zero and therefore delete a live level.
func TestMalformedQuantityIsIgnoredNotTreatedAsDelete(t *testing.T) {
	ob := newInitializedBook(t, snapshot(10,
		[]exchange.PriceLevel{wireLevel("100", "1")},
		[]exchange.PriceLevel{wireLevel("101", "1")},
	))

	ob.HandleDepthUpdate(&exchange.DepthUpdate{
		FinalUpdateID: 11,
		PrevUpdateID:  10,
		Bids:          []exchange.PriceLevel{wireLevel("100", "not-a-number")},
	})

	stats := ob.GetStats()
	if got, want := stats.BestBid.String(), "100"; got != want {
		t.Errorf("BestBid = %s, want %s (level must survive)", got, want)
	}
	if stats.InvalidLevels != 1 {
		t.Errorf("InvalidLevels = %d, want 1", stats.InvalidLevels)
	}
}

// Full-book messages (Hyperliquid, OKX polling) must remove levels that are no
// longer present; merging them left stale liquidity in the book forever.
func TestSnapshotUpdateReplacesBook(t *testing.T) {
	ob := newInitializedBook(t, snapshot(0,
		[]exchange.PriceLevel{wireLevel("100", "1"), wireLevel("99", "1")},
		[]exchange.PriceLevel{wireLevel("101", "1")},
	))

	ob.HandleDepthUpdate(&exchange.DepthUpdate{
		IsSnapshot: true,
		Bids:       []exchange.PriceLevel{wireLevel("100", "5")},
		Asks:       []exchange.PriceLevel{wireLevel("102", "2")},
	})

	stats := ob.GetStats()
	if stats.BidLevels != 1 {
		t.Errorf("BidLevels = %d, want 1 (stale level must be dropped)", stats.BidLevels)
	}
	if stats.AskLevels != 1 {
		t.Errorf("AskLevels = %d, want 1", stats.AskLevels)
	}
	if got, want := stats.BestAsk.String(), "102"; got != want {
		t.Errorf("BestAsk = %s, want %s", got, want)
	}
}

// Updates that were already covered by the snapshot must be ignored instead of
// being reported as a sequence gap. This is the normal Binance startup race.
func TestStaleSequencedUpdateIsIgnored(t *testing.T) {
	ob := newInitializedBook(t, snapshot(100,
		[]exchange.PriceLevel{wireLevel("100", "1")},
		[]exchange.PriceLevel{wireLevel("101", "1")},
	))

	ob.HandleDepthUpdate(&exchange.DepthUpdate{
		FirstUpdateID: 90,
		FinalUpdateID: 99, // u <= lastUpdateId: already in the snapshot
		PrevUpdateID:  89,
		Bids:          []exchange.PriceLevel{wireLevel("100", "0")},
	})

	if ob.NeedsResync() {
		t.Error("stale update must not request a resync")
	}
	stats := ob.GetStats()
	if got, want := stats.TotalBidsQty.String(), "1"; got != want {
		t.Errorf("TotalBidsQty = %s, want %s (stale update must not be applied)", got, want)
	}
}

func TestSequencedGapRequestsResync(t *testing.T) {
	ob := newInitializedBook(t, snapshot(10,
		[]exchange.PriceLevel{wireLevel("100", "1")},
		[]exchange.PriceLevel{wireLevel("101", "1")},
	))

	ob.HandleDepthUpdate(&exchange.DepthUpdate{
		FirstUpdateID: 20,
		FinalUpdateID: 20,
		PrevUpdateID:  19, // expected 10
		Bids:          []exchange.PriceLevel{wireLevel("100", "9")},
	})

	ob.mu.RLock()
	needsResync := ob.needsResync
	buffered := len(ob.eventBuffer)
	ob.mu.RUnlock()

	if !needsResync {
		t.Error("expected needsResync to be set after a sequence gap")
	}
	if buffered != 1 {
		t.Errorf("buffered events = %d, want 1", buffered)
	}

	// The gap must not have been applied to the live book.
	if got, want := ob.GetStats().TotalBidsQty.String(), "1"; got != want {
		t.Errorf("TotalBidsQty = %s, want %s", got, want)
	}
}

func TestProcessBufferedUnsequencedUpdates(t *testing.T) {
	ob := New()
	if err := ob.LoadSnapshot(snapshot(0,
		[]exchange.PriceLevel{wireLevel("100", "1")},
		[]exchange.PriceLevel{wireLevel("101", "1")},
	)); err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}

	// Kraken/Coinbase style updates have no sequence numbers.
	ob.HandleDepthUpdate(&exchange.DepthUpdate{
		Bids: []exchange.PriceLevel{wireLevel("100", "3")},
		Asks: []exchange.PriceLevel{wireLevel("101", "4")},
	})

	ob.ProcessBufferedEvents()

	stats := ob.GetStats()
	if got, want := stats.TotalBidsQty.String(), "3"; got != want {
		t.Errorf("TotalBidsQty = %s, want %s", got, want)
	}
	if got, want := stats.TotalAsksQty.String(), "4"; got != want {
		t.Errorf("TotalAsksQty = %s, want %s", got, want)
	}
}

func TestEmptySideHasZeroBestPrice(t *testing.T) {
	ob := newInitializedBook(t, snapshot(1,
		[]exchange.PriceLevel{wireLevel("100", "1")},
		nil,
	))

	stats := ob.GetStats()
	if !stats.BestAsk.IsZero() {
		t.Errorf("BestAsk = %s, want zero", stats.BestAsk)
	}
	if stats.Spread.String() != "0" {
		t.Errorf("Spread = %s, want 0", stats.Spread)
	}
}

func TestSnapshotSkipsMalformedLevels(t *testing.T) {
	ob := New()
	err := ob.LoadSnapshot(snapshot(1,
		[]exchange.PriceLevel{wireLevel("100", "1"), wireLevel("bad", "1")},
		[]exchange.PriceLevel{wireLevel("101", "1"), wireLevel("102", "-1")},
	))
	if err != nil {
		t.Fatalf("LoadSnapshot returned error: %v", err)
	}

	stats := ob.GetStats()
	if stats.BidLevels != 1 || stats.AskLevels != 1 {
		t.Errorf("levels = %d/%d, want 1/1", stats.BidLevels, stats.AskLevels)
	}
	if stats.InvalidLevels != 2 {
		t.Errorf("InvalidLevels = %d, want 2", stats.InvalidLevels)
	}
}

// Reloading a snapshot (resync or symbol change) must not keep the previous
// best bid/ask when prices moved down.
func TestReloadSnapshotResetsStaleBestPrices(t *testing.T) {
	ob := newInitializedBook(t, snapshot(1,
		[]exchange.PriceLevel{wireLevel("200", "1")},
		[]exchange.PriceLevel{wireLevel("201", "1")},
	))

	if err := ob.LoadSnapshot(snapshot(2,
		[]exchange.PriceLevel{wireLevel("100", "1")},
		[]exchange.PriceLevel{wireLevel("101", "1")},
	)); err != nil {
		t.Fatalf("second LoadSnapshot: %v", err)
	}

	stats := ob.GetStats()
	if got, want := stats.BestBid.String(), "100"; got != want {
		t.Errorf("BestBid after reload = %s, want %s", got, want)
	}
	if got, want := stats.BestAsk.String(), "101"; got != want {
		t.Errorf("BestAsk after reload = %s, want %s", got, want)
	}
}

// Price formatting differences between snapshot and update must not prevent a
// level from being deleted.
func TestDeleteMatchesCanonicalPriceFormat(t *testing.T) {
	ob := newInitializedBook(t, snapshot(10,
		[]exchange.PriceLevel{wireLevel("100.10", "1")},
		[]exchange.PriceLevel{wireLevel("100.20", "1")},
	))

	ob.HandleDepthUpdate(&exchange.DepthUpdate{
		FinalUpdateID: 11,
		PrevUpdateID:  10,
		Bids:          []exchange.PriceLevel{wireLevel("100.1", "0")},
		Asks:          []exchange.PriceLevel{wireLevel("100.2", "0")},
	})

	stats := ob.GetStats()
	if stats.BidLevels != 0 || stats.AskLevels != 0 {
		t.Errorf("levels = %d/%d, want 0/0", stats.BidLevels, stats.AskLevels)
	}
	if !stats.BestBid.IsZero() || !stats.BestAsk.IsZero() {
		t.Errorf("best prices = %s/%s, want 0/0", stats.BestBid, stats.BestAsk)
	}
}
