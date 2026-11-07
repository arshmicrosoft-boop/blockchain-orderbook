# Crypto Orderbook — Frontend

React 19 + TypeScript + Vite + Tailwind frontend for the multi-exchange
orderbook backend.

## Requirements

- Node 20.19+ or 22.12+ (or Bun)
- The Go backend running (default: `ws://localhost:8086/ws`)

## Commands

```bash
bun install      # or npm install
bun run dev      # start Vite dev server
bun run build    # type-check + production build
bun run lint     # ESLint
bun run preview  # preview the production build
```

## Configuration

The app connects to the current page's hostname on port `8086` by default.
Override the full URL with `VITE_WS_URL`:

```bash
VITE_WS_URL=wss://orderbook.example.com/ws bun run build
```

## Structure

- `src/hooks/useWebSocket.ts` — WebSocket connection, reconnect/backoff, message handling
- `src/hooks/useAggregatedOrderbook.ts` — nets bids/asks across the selected exchanges
- `src/hooks/useChartData.ts` — converts stats into chart series
- `src/components/OrderbookCard.tsx` — per-exchange / aggregated depth card
- `src/components/StatsTable.tsx` — sortable exchange statistics table
- `src/components/LiquidityChart.tsx` — liquidity bar charts
- `src/utils/calculations.ts` — filtering, sorting, formatting helpers
