package coinbase

// Config holds configuration for Coinbase exchange
type Config struct {
	Symbol string
}

// SubscribeRequest represents a subscription request to Coinbase WebSocket
type SubscribeRequest struct {
	Type       string   `json:"type"`
	ProductIDs []string `json:"product_ids"`
	Channel    string   `json:"channel"`
}

// WSMessage represents a WebSocket message from Coinbase
type WSMessage struct {
	Channel   string  `json:"channel"`
	Timestamp string  `json:"timestamp"`
	Events    []Event `json:"events"`
}

// Event represents an event in the WebSocket message
type Event struct {
	Type      string   `json:"type"` // "snapshot" or "update"
	ProductID string   `json:"product_id"`
	Updates   []Update `json:"updates"`
}

// Update represents a single price level update
type Update struct {
	Side        string `json:"side"`         // "bid" or "ask"
	EventTime   string `json:"event_time"`   // timestamp
	PriceLevel  string `json:"price_level"`  // price
	NewQuantity string `json:"new_quantity"` // quantity (if "0", remove level)
}
