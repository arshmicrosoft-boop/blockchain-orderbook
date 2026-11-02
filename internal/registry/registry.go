// Package registry provides a concurrency-safe collection of live orderbooks.
//
// The exchange supervisor writes orderbooks into the collection while the
// websocket server iterates it to build messages. Before this registry existed
// both sides shared a plain map, which is a data race that Go turns into a
// fatal "concurrent map iteration and map write" crash.
package registry

import (
	"sort"
	"sync"

	"orderbook/internal/orderbook"
)

// Registry stores orderbooks keyed by exchange name.
type Registry struct {
	mu    sync.RWMutex
	books map[string]*orderbook.OrderBook
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{
		books: make(map[string]*orderbook.OrderBook),
	}
}

// Set stores (or replaces) the orderbook for an exchange.
func (r *Registry) Set(name string, ob *orderbook.OrderBook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.books[name] = ob
}

// Delete removes an exchange's orderbook.
func (r *Registry) Delete(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.books, name)
}

// Get returns the orderbook for an exchange, if present.
func (r *Registry) Get(name string) (*orderbook.OrderBook, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ob, ok := r.books[name]
	return ob, ok
}

// Len returns the number of tracked orderbooks.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.books)
}

// Snapshot returns a copy of the registry contents. Callers may iterate the
// returned map without holding the registry lock.
func (r *Registry) Snapshot() map[string]*orderbook.OrderBook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[string]*orderbook.OrderBook, len(r.books))
	for name, ob := range r.books {
		out[name] = ob
	}
	return out
}

// Names returns the tracked exchange names in sorted order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	names := make([]string, 0, len(r.books))
	for name := range r.books {
		names = append(names, name)
	}
	r.mu.RUnlock()

	sort.Strings(names)
	return names
}
