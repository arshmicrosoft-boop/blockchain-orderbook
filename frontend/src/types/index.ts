// WebSocket message types
export type OrderbookMessage = {
  type: 'orderbook';
  exchange: string;
  bids: Array<{ price: string; quantity: string; cumulative: string }>;
  asks: Array<{ price: string; quantity: string; cumulative: string }>;
  timestamp: number;
};

export type StatsMessage = {
  type: 'stats';
  exchange: string;
  bestBid: string;
  bestAsk: string;
  midPrice: string;
  spread: string;
  bidLiquidity05Pct: string;
  askLiquidity05Pct: string;
  deltaLiquidity05Pct: string;
  bidLiquidity2Pct: string;
  askLiquidity2Pct: string;
  deltaLiquidity2Pct: string;
  bidLiquidity10Pct: string;
  askLiquidity10Pct: string;
  deltaLiquidity10Pct: string;
  totalBidsQty: string;
  totalAsksQty: string;
  totalDelta: string;
  timestamp: number;
};

export type WebSocketMessage = OrderbookMessage | StatsMessage;

// Data structures
export type OrderbookLevel = {
  price: string;
  quantity: string;
  cumulative: string;
};

export type OrderbookData = {
  [exchange: string]: {
    bids: OrderbookLevel[];
    asks: OrderbookLevel[];
  };
};

export type StatsData = {
  [exchange: string]: Omit<StatsMessage, 'type' | 'exchange' | 'timestamp'>;
};

// UI types
export type MarketFilter = 'all' | 'spot' | 'perps';

export type TickLevel = {
  value: string;
  label: string;
};

// Chart data types
export type ChartDataPoint = {
  exchange: string;
  bid: number;
  ask: number;
};