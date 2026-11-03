package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"orderbook/internal/aggregation"
	"orderbook/internal/orderbook"
	"orderbook/internal/registry"
	"orderbook/internal/types"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

type MessageType string

const (
	MessageTypeOrderbook MessageType = "orderbook"
	MessageTypeStats     MessageType = "stats"
)

const (
	pushInterval = 200 * time.Millisecond

	// writeWait bounds a single write so one stalled client cannot pin a
	// goroutine (or the broadcaster) forever.
	writeWait = 5 * time.Second
	// pongWait must be larger than pingPeriod.
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second

	maxMessageSize = 4 * 1024
	clientSendBuf  = 64
)

// ClientMessage represents messages sent from client to server.
type ClientMessage struct {
	Type   string  `json:"type"`
	Tick   float64 `json:"tick,omitempty"`
	Symbol string  `json:"symbol,omitempty"`
}

type OrderbookMessage struct {
	Type      MessageType  `json:"type"`
	Exchange  string       `json:"exchange"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp int64        `json:"timestamp"`
}

type StatsMessage struct {
	Type                MessageType `json:"type"`
	Exchange            string      `json:"exchange"`
	BestBid             string      `json:"bestBid"`
	BestAsk             string      `json:"bestAsk"`
	MidPrice            string      `json:"midPrice"`
	Spread              string      `json:"spread"`
	BidLiquidity05Pct   string      `json:"bidLiquidity05Pct"`
	AskLiquidity05Pct   string      `json:"askLiquidity05Pct"`
	DeltaLiquidity05Pct string      `json:"deltaLiquidity05Pct"`
	BidLiquidity2Pct    string      `json:"bidLiquidity2Pct"`
	AskLiquidity2Pct    string      `json:"askLiquidity2Pct"`
	DeltaLiquidity2Pct  string      `json:"deltaLiquidity2Pct"`
	BidLiquidity10Pct   string      `json:"bidLiquidity10Pct"`
	AskLiquidity10Pct   string      `json:"askLiquidity10Pct"`
	DeltaLiquidity10Pct string      `json:"deltaLiquidity10Pct"`
	TotalBidsQty        string      `json:"totalBidsQty"`
	TotalAsksQty        string      `json:"totalAsksQty"`
	TotalDelta          string      `json:"totalDelta"`
	SequenceGaps        int64       `json:"sequenceGaps"`
	InvalidLevels       int64       `json:"invalidLevels"`
	Timestamp           int64       `json:"timestamp"`
}

type PriceLevel struct {
	Price      string `json:"price"`
	Quantity   string `json:"quantity"`
	Cumulative string `json:"cumulative"`
}

// client owns one websocket connection. All writes happen on its writePump
// goroutine, so a slow client can never block the broadcaster.
type client struct {
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
	once sync.Once
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.done)
	})
}

// enqueue adds a payload to the client's send queue. If the queue is full the
// oldest frame is dropped: every orderbook frame is a full snapshot, so
// dropping an old one is safe and keeps latency bounded.
func (c *client) enqueue(payload []byte) {
	select {
	case <-c.done:
		return
	default:
	}

	select {
	case c.send <- payload:
	default:
		select {
		case <-c.send:
		default:
		}
		select {
		case c.send <- payload:
		default:
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.done:
			return
		case payload := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type Server struct {
	orderbooks   *registry.Registry
	port         string
	upgrader     websocket.Upgrader
	clientsMu    sync.RWMutex
	clients      map[*client]struct{}
	broadcast    chan []byte
	aggregator   *aggregation.Aggregator
	tickMux      sync.RWMutex
	symbolChange chan string
	maxDepth     int
}

// NewServer creates a websocket server.
//
// maxDepth caps the number of price levels sent per side. The frontend only
// renders a handful; streaming entire 5000-level books for every exchange five
// times per second wasted tens of megabytes per second.
func NewServer(books *registry.Registry, port string, symbolChange chan string, maxDepth int) *Server {
	return &Server{
		orderbooks:   books,
		port:         port,
		clients:      make(map[*client]struct{}),
		broadcast:    make(chan []byte, 512),
		aggregator:   aggregation.New(types.Tick1), // Default to 1.0 tick
		symbolChange: symbolChange,
		maxDepth:     maxDepth,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin:     checkOrigin,
		},
	}
}

// extraAllowedOrigins lets a deployment serve the frontend from a different
// origin (for example an SPA host) without weakening the default: only the
// exact origins listed in ALLOWED_ORIGINS are accepted in addition to
// same-host and loopback origins.
var extraAllowedOrigins = parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))

func parseAllowedOrigins(raw string) map[string]struct{} {
	origins := make(map[string]struct{})
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}
	return origins
}

// checkOrigin only accepts same-host, loopback, and explicitly allow-listed
// origins. The previous implementation accepted every origin, which allows any
// website to read the local orderbook feed (DNS-rebinding style access).
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients do not send an Origin header.
		return true
	}

	if _, ok := extraAllowedOrigins[origin]; ok {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}

	return u.Host == r.Host
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	go s.broadcastMessages()
	go s.startDataPush()

	srv := &http.Server{
		Addr:              ":" + s.port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("WebSocket server starting on port %s", s.port)
	return srv.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.clientsMu.RLock()
	clients := len(s.clients)
	s.clientsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"exchanges": s.orderbooks.Len(),
		"clients":   clients,
		"maxDepth":  s.maxDepth,
	})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	c := &client{
		conn: conn,
		send: make(chan []byte, clientSendBuf),
		done: make(chan struct{}),
	}

	s.clientsMu.Lock()
	s.clients[c] = struct{}{}
	s.clientsMu.Unlock()

	log.Printf("New WebSocket client connected from %s", r.RemoteAddr)

	go c.writePump()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, c)
		s.clientsMu.Unlock()
		c.close()
		conn.Close()
		log.Printf("WebSocket client disconnected")
	}()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Printf("Error parsing client message: %v", err)
			continue
		}

		s.handleClientMessage(clientMsg)
	}
}

func (s *Server) handleClientMessage(msg ClientMessage) {
	switch msg.Type {
	case "set_tick":
		s.setTickLevel(msg.Tick)
	case "change_symbol":
		if msg.Symbol == "" {
			return
		}
		// Never block a client's reader goroutine on the supervisor loop.
		select {
		case s.symbolChange <- msg.Symbol:
			log.Printf("Symbol change request: %s", msg.Symbol)
		default:
			log.Printf("Symbol change to %s ignored: another change is still pending", msg.Symbol)
		}
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

func (s *Server) setTickLevel(tick float64) {
	tickLevel := types.TickLevel(tick)

	validTick := false
	for _, available := range types.AvailableTickLevels {
		if available == tickLevel {
			validTick = true
			break
		}
	}

	if !validTick {
		log.Printf("Invalid tick level: %f, ignoring", tick)
		return
	}

	s.tickMux.Lock()
	s.aggregator.SetTickLevel(tickLevel)
	s.tickMux.Unlock()

	log.Printf("Tick level changed to: %f", tick)
}

func (s *Server) broadcastMessages() {
	for payload := range s.broadcast {
		s.clientsMu.RLock()
		for c := range s.clients {
			c.enqueue(payload)
		}
		s.clientsMu.RUnlock()
	}
}

// push queues an already-marshalled message for fan-out. If the queue is full
// the oldest entry is dropped; values are periodic full snapshots.
func (s *Server) push(payload []byte) {
	select {
	case s.broadcast <- payload:
	default:
		select {
		case <-s.broadcast:
		default:
		}
		select {
		case s.broadcast <- payload:
		default:
		}
	}
}

func (s *Server) startDataPush() {
	ticker := time.NewTicker(pushInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.clientsMu.RLock()
		hasClients := len(s.clients) > 0
		s.clientsMu.RUnlock()

		if !hasClients {
			continue
		}

		timestamp := time.Now().UnixMilli()

		for exchangeName, ob := range s.orderbooks.Snapshot() {
			if !ob.IsInitialized() {
				continue
			}

			if payload, err := json.Marshal(s.buildOrderbookMessage(exchangeName, ob, timestamp)); err == nil {
				s.push(payload)
			} else {
				log.Printf("Failed to marshal orderbook for %s: %v", exchangeName, err)
			}

			if payload, err := json.Marshal(s.buildStatsMessage(exchangeName, ob, timestamp)); err == nil {
				s.push(payload)
			} else {
				log.Printf("Failed to marshal stats for %s: %v", exchangeName, err)
			}
		}
	}
}

func (s *Server) buildOrderbookMessage(exchange string, ob *orderbook.OrderBook, timestamp int64) OrderbookMessage {
	bidLevels, askLevels := ob.SnapshotLevels()

	s.tickMux.RLock()
	aggregatedBids := s.aggregator.AggregateBids(bidLevels)
	aggregatedAsks := s.aggregator.AggregateAsks(askLevels)
	s.tickMux.RUnlock()

	sort.Slice(aggregatedBids, func(i, j int) bool {
		return aggregatedBids[i].Price.GreaterThan(aggregatedBids[j].Price)
	})
	sort.Slice(aggregatedAsks, func(i, j int) bool {
		return aggregatedAsks[i].Price.LessThan(aggregatedAsks[j].Price)
	})

	if s.maxDepth > 0 {
		aggregatedBids = truncate(aggregatedBids, s.maxDepth)
		aggregatedAsks = truncate(aggregatedAsks, s.maxDepth)
	}

	bids := make([]PriceLevel, 0, len(aggregatedBids))
	bidCumulative := decimal.Zero
	for _, bid := range aggregatedBids {
		bidCumulative = bidCumulative.Add(bid.Quantity)
		bids = append(bids, PriceLevel{
			Price:      bid.Price.String(),
			Quantity:   bid.Quantity.String(),
			Cumulative: bidCumulative.String(),
		})
	}

	asks := make([]PriceLevel, 0, len(aggregatedAsks))
	askCumulative := decimal.Zero
	for _, ask := range aggregatedAsks {
		askCumulative = askCumulative.Add(ask.Quantity)
		asks = append(asks, PriceLevel{
			Price:      ask.Price.String(),
			Quantity:   ask.Quantity.String(),
			Cumulative: askCumulative.String(),
		})
	}

	return OrderbookMessage{
		Type:      MessageTypeOrderbook,
		Exchange:  exchange,
		Bids:      bids,
		Asks:      asks,
		Timestamp: timestamp,
	}
}

func truncate(levels []types.PriceLevel, max int) []types.PriceLevel {
	if len(levels) <= max {
		return levels
	}
	return levels[:max]
}

func (s *Server) buildStatsMessage(exchange string, ob *orderbook.OrderBook, timestamp int64) StatsMessage {
	stats := ob.GetStats()

	midPrice := decimal.Zero
	if !stats.BestBid.IsZero() && !stats.BestAsk.IsZero() {
		midPrice = stats.BestBid.Add(stats.BestAsk).Div(decimal.NewFromInt(2))
	}

	return StatsMessage{
		Type:                MessageTypeStats,
		Exchange:            exchange,
		BestBid:             stats.BestBid.String(),
		BestAsk:             stats.BestAsk.String(),
		MidPrice:            midPrice.String(),
		Spread:              stats.Spread.String(),
		BidLiquidity05Pct:   stats.BidLiquidity05Pct.String(),
		AskLiquidity05Pct:   stats.AskLiquidity05Pct.String(),
		DeltaLiquidity05Pct: stats.DeltaLiquidity05Pct.String(),
		BidLiquidity2Pct:    stats.BidLiquidity2Pct.String(),
		AskLiquidity2Pct:    stats.AskLiquidity2Pct.String(),
		DeltaLiquidity2Pct:  stats.DeltaLiquidity2Pct.String(),
		BidLiquidity10Pct:   stats.BidLiquidity10Pct.String(),
		AskLiquidity10Pct:   stats.AskLiquidity10Pct.String(),
		DeltaLiquidity10Pct: stats.DeltaLiquidity10Pct.String(),
		TotalBidsQty:        stats.TotalBidsQty.String(),
		TotalAsksQty:        stats.TotalAsksQty.String(),
		TotalDelta:          stats.TotalDelta.String(),
		SequenceGaps:        stats.SequenceGaps,
		InvalidLevels:       stats.InvalidLevels,
		Timestamp:           timestamp,
	}
}
