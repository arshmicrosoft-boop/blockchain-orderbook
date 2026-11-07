# Blockchain Orderbook

Multi-exchange, real-time orderbook with a Go backend and a React + Vite frontend.

> **Note:** This is an experimental project built quickly for learning and prototyping. Expect rough edges.
---

What it is
- Backend: Go service that connects to several exchanges, maintains live orderbooks, and serves a WebSocket feed at ws://localhost:8086/ws. It also exposes a small HTTP health endpoint at http://localhost:8086/health. See [cmd/main.go](cmd/main.go) and [internal/websocket/server.go](internal/websocket/server.go).
- Frontend: React app that subscribes to the WebSocket, renders per-exchange orderbooks, aggregated orderbooks, statistics, and charts. See [frontend/src/App.tsx](frontend/src/App.tsx).

Project layout
- Go backend (exchanges, orderbook engine, websocket):
  - [cmd/main.go](cmd/main.go)
  - [internal/exchange](internal/exchange)
  - [internal/orderbook](internal/orderbook)
  - [internal/registry/registry.go](internal/registry/registry.go)
  - [internal/websocket/server.go](internal/websocket/server.go)
- Frontend (React + Vite + Tailwind):
  - [frontend](frontend)
  - Entry: [frontend/src/main.tsx](frontend/src/main.tsx)
  - App: [frontend/src/App.tsx](frontend/src/App.tsx)

Quick start

Backend (Go 1.24+)
```bash
# Defaults: BTCUSDT, port 8086, 100 levels per side
go run ./cmd

# Or with explicit options
go run ./cmd -symbol BTCUSDT -ws-port 8086 -depth 100 -log-interval 10s
```

Frontend (Node 20.19+ / 22.12+, or Bun)
```bash
cd frontend
bun install   # or npm install
bun run dev   # or npm run dev
# Open the URL printed by Vite http://localhost:5173
```

Backend flags
| Flag | Default | Meaning |
|------|---------|---------|
| `-symbol` | `BTCUSDT` | Trading pair to monitor |
| `-ws-port` | `8086` | WebSocket and HTTP port |
| `-depth` | `100` | Max price levels sent per side (`0` = unlimited) |
| `-log-interval` | `10s` | How often orderbook stats are printed to the console |

How it works
- The backend starts a WebSocket server at ws://localhost:8086/ws and streams:
  - orderbook messages per exchange (bids/asks levels, capped at `-depth`)
  - stats messages per exchange (best bid/ask, spread, liquidity at 0.5%, 2%, 10%, totals, sequence-gap/invalid-level counters)
- The frontend connects to ws://localhost:8086/ws and renders:
  - Exchange Statistics table
  - Individual Order Books or an Aggregated Order Book
  - Liquidity charts (0.5%, 2%, 10%, total)
  - Market filter (All / Spot / Perps), tick size selector, dark/light theme

The frontend defaults to the page's hostname on port 8086. Set `VITE_WS_URL` to override it, for example:
```bash
VITE_WS_URL=wss://orderbook.example.com/ws bun run build
```

Resilience behavior
- Each exchange reconnects automatically with exponential backoff (1s → 30s) if its stream drops.
- The orderbook engine detects sequence gaps and re-syncs by reconnecting for a fresh snapshot instead of silently applying corrupt updates.
- Full-book messages (Hyperliquid, OKX polling, exchange resnapshots) replace the local book rather than being merged as deltas.
- Malformed price levels are skipped and counted, never treated as zero-quantity deletions.
- Slow WebSocket clients are isolated: each client has its own write queue, and stale frames are dropped instead of blocking the broadcaster.
- `/health` returns JSON with the connected exchange count, client count, and configured depth cap.
- WebSocket origins default to same-host and loopback only. If the frontend is served from a different origin, list it in `ALLOWED_ORIGINS` (comma-separated), e.g. `ALLOWED_ORIGINS=https://app.example.com go run ./cmd`.

Controls (frontend)
- Market filter: All, Spot, Perps (top-right toggle)
- Theme: dark/light toggle
- Tick: select the aggregation step (e.g., 0.1, 1, 10, 50, 100)
- Aggregate: toggle between per-exchange and aggregated orderbook views

Exchanges enabled
- The backend is configured in [cmd/main.go](cmd/main.go) to connect to:
  - Binance (spot), Binancef (perps)
  - Bybit (spot), Bybitf (perps)
  - Kraken (spot)
  - OKX (spot)
  - Coinbase (spot)
  - Asterdexf (perps)
  - BingX (spot)
  - Hyperliquidf (perps)

Builds

Backend
```bash
# Build a binary
go build -o crypto-orderbook ./cmd

# Run tests
go test ./...

# Run with the race detector (requires cgo and a C toolchain)
go run -race ./cmd
```

Frontend
```bash
cd frontend
bun run build     # or npm run build
bun run preview   # or npm run preview
bun run lint      # or npm run lint
```

Notes
- The frontend connects to the page's hostname on port 8086 by default. Set `VITE_WS_URL` if the backend runs elsewhere (see [frontend/src/App.tsx](frontend/src/App.tsx)).
- If you change the WebSocket server port, update `VITE_WS_URL` or the default URL in [frontend/src/App.tsx](frontend/src/App.tsx).
