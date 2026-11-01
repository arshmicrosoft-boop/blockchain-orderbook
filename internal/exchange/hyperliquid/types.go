package hyperliquid

// L2BookResponse represents the REST API response for Hyperliquid L2 book snapshot
type L2BookResponse struct {
	Coin   string       `json:"coin"`
	Time   int64        `json:"time"`
	Levels [2][]WsLevel `json:"levels"` // [bids[], asks[]]
}

// WsBook represents the WebSocket L2 book update from Hyperliquid
type WsBook struct {
	Coin   string       `json:"coin"`
	Time   int64        `json:"time"`
	Levels [2][]WsLevel `json:"levels"` // [bids[], asks[]]
}

// WsLevel represents a single price level in Hyperliquid format
type WsLevel struct {
	Px string `json:"px"` // price
	Sz string `json:"sz"` // size
	N  int    `json:"n"`  // number of orders
}

// SubscriptionMessage represents the WebSocket subscription message
type SubscriptionMessage struct {
	Method       string                 `json:"method"`
	Subscription map[string]interface{} `json:"subscription"`
}

// SubscriptionResponse represents the WebSocket subscription acknowledgment
type SubscriptionResponse struct {
	Channel string                 `json:"channel"`
	Data    map[string]interface{} `json:"data"`
}

// WSMessage represents a generic WebSocket message
type WSMessage struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}