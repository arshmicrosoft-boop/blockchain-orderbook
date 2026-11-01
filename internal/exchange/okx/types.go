package okx

// Config holds configuration for OKX exchange
type Config struct {
	Symbol string
}

// OrderBookResponse represents the REST API response for OKX order book
type OrderBookResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data []OrderBookData `json:"data"`
}

// OrderBookData represents the orderbook data in the REST response
type OrderBookData struct {
	Asks [][]string `json:"asks"` // [price, quantity, deprecated, order_count]
	Bids [][]string `json:"bids"` // [price, quantity, deprecated, order_count]
	Ts   string     `json:"ts"`   // timestamp
}
