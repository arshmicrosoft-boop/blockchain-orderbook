package binance

// SnapshotResponse represents the REST API response for Binance order book snapshot
type SnapshotResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// WSMessage represents a WebSocket message from Binance
type WSMessage struct {
	Stream string      `json:"stream"`
	Data   DepthUpdate `json:"data"`
}

// DepthUpdate represents a depth update event from Binance WebSocket
type DepthUpdate struct {
	EventType     string     `json:"e"`
	EventTime     int64      `json:"E"`
	Symbol        string     `json:"s"`
	FirstUpdateID int64      `json:"U"`
	FinalUpdateID int64      `json:"u"`
	PrevUpdateID  int64      `json:"pu"`
	Bids          [][]string `json:"b"`
	Asks          [][]string `json:"a"`
}
