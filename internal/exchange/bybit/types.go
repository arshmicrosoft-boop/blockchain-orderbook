package bybit

// WSMessage represents a WebSocket message from Bybit
type WSMessage struct {
	Topic string        `json:"topic"`
	Type  string        `json:"type"` // "snapshot" or "delta"
	TS    int64         `json:"ts"`
	Data  OrderbookData `json:"data"`
	CTS   int64         `json:"cts"` // matching engine timestamp
}

// OrderbookData represents the orderbook data from Bybit
type OrderbookData struct {
	Symbol   string     `json:"s"`
	Bids     [][]string `json:"b"` // [price, size]
	Asks     [][]string `json:"a"` // [price, size]
	UpdateID int64      `json:"u"`
	SeqNum   int64      `json:"seq"`
}

// SubscribeMessage represents a subscription request
type SubscribeMessage struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}
