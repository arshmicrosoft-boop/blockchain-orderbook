package websocket

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"orderbook/internal/exchange"
	"orderbook/internal/orderbook"
	"orderbook/internal/registry"
)

func newTestBook(t *testing.T) *orderbook.OrderBook {
	t.Helper()

	ob := orderbook.New()
	err := ob.LoadSnapshot(&exchange.Snapshot{
		Bids: []exchange.PriceLevel{
			{Price: "100", Quantity: "1"},
			{Price: "99", Quantity: "2"},
			{Price: "98", Quantity: "3"},
		},
		Asks: []exchange.PriceLevel{
			{Price: "101", Quantity: "1"},
			{Price: "102", Quantity: "2"},
			{Price: "103", Quantity: "3"},
		},
	})
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	ob.ProcessBufferedEvents()
	return ob
}

func newTestServer(maxDepth int) *Server {
	return NewServer(registry.New(), "8086", make(chan string, 1), maxDepth)
}

func TestBuildOrderbookMessageOrdersAndTruncates(t *testing.T) {
	s := newTestServer(2)
	msg := s.buildOrderbookMessage("binance", newTestBook(t), 123)

	if len(msg.Bids) != 2 {
		t.Fatalf("bids = %d, want 2", len(msg.Bids))
	}
	if len(msg.Asks) != 2 {
		t.Fatalf("asks = %d, want 2", len(msg.Asks))
	}
	if msg.Bids[0].Price != "100" || msg.Bids[1].Price != "99" {
		t.Errorf("bids ordered incorrectly: %+v", msg.Bids)
	}
	if msg.Asks[0].Price != "101" || msg.Asks[1].Price != "102" {
		t.Errorf("asks ordered incorrectly: %+v", msg.Asks)
	}
	if msg.Bids[0].Cumulative != "1" || msg.Bids[1].Cumulative != "3" {
		t.Errorf("bid cumulative = %s, %s; want 1, 3", msg.Bids[0].Cumulative, msg.Bids[1].Cumulative)
	}
	if msg.Asks[0].Cumulative != "1" || msg.Asks[1].Cumulative != "3" {
		t.Errorf("ask cumulative = %s, %s; want 1, 3", msg.Asks[0].Cumulative, msg.Asks[1].Cumulative)
	}
}

func TestBuildOrderbookMessageUnlimitedDepth(t *testing.T) {
	s := newTestServer(0)
	msg := s.buildOrderbookMessage("binance", newTestBook(t), 0)

	if len(msg.Bids) != 3 || len(msg.Asks) != 3 {
		t.Fatalf("levels = %d/%d, want 3/3", len(msg.Bids), len(msg.Asks))
	}
}

func TestBuildStatsMessageMidPriceGuarded(t *testing.T) {
	s := newTestServer(10)
	empty := orderbook.New()
	empty.ProcessBufferedEvents()

	msg := s.buildStatsMessage("binance", empty, 0)
	if msg.MidPrice != "0" {
		t.Errorf("MidPrice = %s, want 0 when the book is empty", msg.MidPrice)
	}
}

func TestEnqueueDropsOldestWhenFull(t *testing.T) {
	c := &client{
		send: make(chan []byte, 2),
		done: make(chan struct{}),
	}

	c.enqueue([]byte("a"))
	c.enqueue([]byte("b"))
	c.enqueue([]byte("c"))

	first := <-c.send
	second := <-c.send
	if string(first) != "b" || string(second) != "c" {
		t.Errorf("queue = %q, %q; want b, c", first, second)
	}
}

func TestParseAllowedOrigins(t *testing.T) {
	got := parseAllowedOrigins(" https://app.test ,https://other.test, ,")
	if _, ok := got["https://app.test"]; !ok {
		t.Error("missing https://app.test")
	}
	if _, ok := got["https://other.test"]; !ok {
		t.Error("missing https://other.test")
	}
	if len(got) != 2 {
		t.Errorf("origins = %v, want 2 entries", got)
	}
}

func TestCheckOrigin(t *testing.T) {	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{name: "no origin", host: "localhost:8086", want: true},
		{name: "localhost dev server", host: "localhost:8086", origin: "http://localhost:5173", want: true},
		{name: "loopback", host: "127.0.0.1:8086", origin: "http://127.0.0.1:5173", want: true},
		{name: "same host", host: "example.test", origin: "http://example.test", want: true},
		{name: "foreign origin", host: "localhost:8086", origin: "https://evil.test", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/ws", nil)
			r.Host = tt.host
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if got := checkOrigin(r); got != tt.want {
				t.Errorf("checkOrigin = %v, want %v", got, tt.want)
			}
		})
	}
}

// The wire format must stay stable: the frontend depends on these field names.
func TestMessageJSONShape(t *testing.T) {
	s := newTestServer(1)
	payload, err := json.Marshal(s.buildOrderbookMessage("binance", newTestBook(t), 7))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"type", "exchange", "bids", "asks", "timestamp"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("missing key %q in %s", key, payload)
		}
	}
}
