package kraken

// Config holds configuration for Kraken exchange
type Config struct {
	Symbol string
}

// SubscribeRequest represents a subscription request to Kraken WebSocket v2
type SubscribeRequest struct {
	Method string          `json:"method"`
	Params SubscribeParams `json:"params"`
	ReqID  int             `json:"req_id,omitempty"`
}

// SubscribeParams holds the subscription parameters
type SubscribeParams struct {
	Channel  string   `json:"channel"`
	Symbol   []string `json:"symbol"`
	Depth    int      `json:"depth"`
	Snapshot bool     `json:"snapshot"`
}

// SubscribeResponse represents the subscription acknowledgement
type SubscribeResponse struct {
	Method  string          `json:"method"`
	Result  SubscribeResult `json:"result"`
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	TimeIn  string          `json:"time_in,omitempty"`
	TimeOut string          `json:"time_out,omitempty"`
	ReqID   int             `json:"req_id,omitempty"`
}

// SubscribeResult holds subscription result details
type SubscribeResult struct {
	Channel  string   `json:"channel"`
	Symbol   string   `json:"symbol"`
	Depth    int      `json:"depth"`
	Snapshot bool     `json:"snapshot"`
	Warnings []string `json:"warnings,omitempty"`
}

// WSMessage represents a WebSocket data message from Kraken
type WSMessage struct {
	Channel string     `json:"channel"`
	Type    string     `json:"type"` // "snapshot" or "update"
	Data    []BookData `json:"data"`
}

// BookData represents the orderbook data
type BookData struct {
	Symbol    string     `json:"symbol"`
	Bids      []PriceQty `json:"bids"`
	Asks      []PriceQty `json:"asks"`
	Checksum  int64      `json:"checksum"`
	Timestamp string     `json:"timestamp,omitempty"` // Only present in updates
}

// PriceQty represents a price level with price and quantity
type PriceQty struct {
	Price float64 `json:"price"`
	Qty   float64 `json:"qty"`
}
