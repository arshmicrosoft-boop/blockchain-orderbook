package registry

import (
	"fmt"
	"sync"
	"testing"

	"orderbook/internal/orderbook"
)

func TestSetGetDelete(t *testing.T) {
	r := New()
	ob := orderbook.New()

	if _, ok := r.Get("binance"); ok {
		t.Fatal("expected empty registry")
	}

	r.Set("binance", ob)
	got, ok := r.Get("binance")
	if !ok || got != ob {
		t.Fatalf("expected stored orderbook, got %v ok=%v", got, ok)
	}
	if r.Len() != 1 {
		t.Fatalf("expected len 1, got %d", r.Len())
	}

	r.Delete("binance")
	if _, ok := r.Get("binance"); ok {
		t.Fatal("expected orderbook to be deleted")
	}
}

func TestSnapshotIsACopy(t *testing.T) {
	r := New()
	r.Set("binance", orderbook.New())

	snap := r.Snapshot()
	delete(snap, "binance")

	if r.Len() != 1 {
		t.Fatal("mutating the snapshot must not mutate the registry")
	}
}

func TestNamesAreSorted(t *testing.T) {
	r := New()
	r.Set("okx", orderbook.New())
	r.Set("binance", orderbook.New())
	r.Set("kraken", orderbook.New())

	names := r.Names()
	want := []string{"binance", "kraken", "okx"}
	if len(names) != len(want) {
		t.Fatalf("expected %d names, got %d", len(want), len(names))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, names)
		}
	}
}

// Run with -race to verify the registry is safe for concurrent use.
func TestConcurrentAccess(t *testing.T) {
	r := New()
	var wg sync.WaitGroup

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("exchange-%d", i%4)
			for j := 0; j < 200; j++ {
				r.Set(name, orderbook.New())
				r.Delete(name)
				_ = r.Snapshot()
				_ = r.Names()
			}
		}(i)
	}

	wg.Wait()
}
