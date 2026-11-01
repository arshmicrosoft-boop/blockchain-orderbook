package asterdex

// SnapshotResponse represents the REST API response for Asterdex order book snapshot
type SnapshotResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// DepthUpdate represents a depth update event from Asterdex WebSocket
type DepthUpdate struct {
	EventType       string     `json:"e"`  // Event type
	EventTime       int64      `json:"E"`  // Event time
	TransactionTime int64      `json:"T"`  // Transaction time
	Symbol          string     `json:"s"`  // Symbol
	FirstUpdateID   int64      `json:"U"`  // First update ID in event
	FinalUpdateID   int64      `json:"u"`  // Final update ID in event
	PrevUpdateID    int64      `json:"pu"` // Final update Id in last stream
	Bids            [][]string `json:"b"`  // Bids to be updated
	Asks            [][]string `json:"a"`  // Asks to be updated
}
