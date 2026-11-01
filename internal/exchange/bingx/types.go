package bingx

// Config holds configuration for BingX exchange
type Config struct {
	Symbol string
}

// SubscriptionMessage represents the subscription request to BingX WebSocket
type SubscriptionMessage struct {
	ID       string `json:"id"`
	ReqType  string `json:"reqType"`
	DataType string `json:"dataType"`
}

// WSMessage represents a WebSocket message from BingX
// BingX sends messages as either text or binary (gzip compressed)
type WSMessage struct {
	Code       int          `json:"code,omitempty"`
	Msg        string       `json:"msg,omitempty"`
	DataType   string       `json:"dataType,omitempty"`
	Data       DepthData    `json:"data,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
}

// DepthData represents the depth update data from BingX Spot (map format)
type DepthData struct {
	Action       string            `json:"action"`       // "all" for snapshot, "update" for incremental
	LastUpdateID int64             `json:"lastUpdateId"` // Update ID for tracking continuity
	Bids         map[string]string `json:"bids"`         // map[price]quantity (Spot uses map format)
	Asks         map[string]string `json:"asks"`         // map[price]quantity (Spot uses map format)
}

// FuturesDepthData represents the depth update data from BingX Futures (array format)
type FuturesDepthData struct {
	Action       string     `json:"action"`       // "all" for snapshot, "update" for incremental
	LastUpdateID int64      `json:"lastUpdateId"` // Update ID for tracking continuity
	Bids         [][]string `json:"bids"`         // [["price", "quantity"]] (Futures uses array format)
	Asks         [][]string `json:"asks"`         // [["price", "quantity"]] (Futures uses array format)
	Time         int64      `json:"time"`         // Timestamp
}

// FuturesWSMessage represents a WebSocket message from BingX Futures
type FuturesWSMessage struct {
	Code       int              `json:"code,omitempty"`
	Msg        string           `json:"msg,omitempty"`
	DataType   string           `json:"dataType,omitempty"`
	Data       FuturesDepthData `json:"data,omitempty"`
	Timestamp  int64            `json:"ts,omitempty"`
}
