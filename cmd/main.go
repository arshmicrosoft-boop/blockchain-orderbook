package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"orderbook/internal/exchange"
	"orderbook/internal/factory"
	"orderbook/internal/orderbook"
	"orderbook/internal/registry"
	"orderbook/internal/websocket"

	"github.com/shopspring/decimal"
)

func main() {
	var symbol = flag.String("symbol", "BTCUSDT", "Trading symbol to monitor")
	var logInterval = flag.Duration("log-interval", 10*time.Second, "Interval for logging orderbook stats")
	var wsPort = flag.String("ws-port", "8086", "WebSocket/HTTP port")
	var maxDepth = flag.Int("depth", 100, "Maximum number of price levels sent per side (0 = unlimited)")
	flag.Parse()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log.Printf("Starting multi-exchange orderbook monitor for %s", *symbol)
	log.Printf("Log interval: %v | Port: %s | Depth: %d", *logInterval, *wsPort, *maxDepth)

	runMultiExchange(*symbol, *logInterval, *wsPort, *maxDepth, interrupt)
}

const (
	colorReset   = "\033[0m"
	colorYellow  = "\033[33m"
	colorGreen   = "\033[32m"
	colorRed     = "\033[31m"
	colorMagenta = "\033[35m"
	colorBold    = "\033[1m"

	reinitCheckInterval = 5 * time.Second
	maxReconnectBackoff = 30 * time.Second
)

func getExchangeNames() []exchange.ExchangeName {
	return []exchange.ExchangeName{
		exchange.Binancef,
		exchange.Binance,
		exchange.Bybitf,
		exchange.Bybit,
		exchange.Kraken,
		exchange.OKX,
		exchange.Coinbase,
		exchange.Asterdexf,
		exchange.BingX,
		exchange.Hyperliquidf,
	}
}

func runMultiExchange(initialSymbol string, logInterval time.Duration, wsPort string, maxDepth int, interrupt chan os.Signal) {
	ctx := context.Background()
	books := registry.New()
	symbolChange := make(chan string, 1)
	currentSymbol := initialSymbol

	wsServer := websocket.NewServer(books, wsPort, symbolChange, maxDepth)
	go func() {
		if err := wsServer.Start(); err != nil {
			log.Fatalf("WebSocket server error: %v", err)
		}
	}()

	for {
		log.Printf("Starting exchanges for symbol: %s", currentSymbol)

		done := make(chan struct{})
		exchangesDone := make(chan struct{})

		go func() {
			superviseExchanges(ctx, currentSymbol, books, logInterval, done, interrupt)
			close(exchangesDone)
		}()

		select {
		case newSymbol := <-symbolChange:
			log.Printf("Symbol change requested: %s -> %s", currentSymbol, newSymbol)
			currentSymbol = newSymbol

			close(done)
			<-exchangesDone

			log.Printf("All exchanges stopped. Restarting with symbol: %s", currentSymbol)
			time.Sleep(500 * time.Millisecond)

		case <-interrupt:
			log.Println("Interrupt received, shutting down...")
			close(done)
			<-exchangesDone
			log.Println("All exchanges closed. Goodbye!")
			return
		}
	}
}

func superviseExchanges(ctx context.Context, symbol string, books *registry.Registry, logInterval time.Duration, done chan struct{}, interrupt chan os.Signal) {
	names := getExchangeNames()

	var wg sync.WaitGroup
	wg.Add(len(names))
	for _, name := range names {
		go func(name exchange.ExchangeName) {
			defer wg.Done()
			runExchange(ctx, name, symbol, books, done, interrupt)
		}(name)
	}

	logDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(logInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				printCombinedStats(books)
			case <-logDone:
				return
			}
		}
	}()

	wg.Wait()
	close(logDone)
}

// runExchange keeps a single exchange connected. If the stream drops it
// reconnects with exponential backoff instead of silently disappearing from the
// dashboard, which is what the old implementation did.
func runExchange(ctx context.Context, name exchange.ExchangeName, symbol string, books *registry.Registry, done <-chan struct{}, interrupt <-chan os.Signal) {
	backoff := time.Second

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			return
		default:
		}

		ex, err := factory.NewExchange(factory.ExchangeConfig{
			Name:   name,
			Symbol: symbol,
		})
		if err != nil {
			log.Printf("[%s] Failed to create exchange: %v", name, err)
			return
		}

		if err := ex.Connect(ctx); err != nil {
			log.Printf("[%s] Failed to connect: %v", name, err)
			_ = ex.Close()
			if !wait(backoff, done, interrupt) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		snapshot, err := ex.GetSnapshot(ctx)
		if err != nil {
			log.Printf("[%s] Failed to get snapshot: %v", name, err)
			_ = ex.Close()
			if !wait(backoff, done, interrupt) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		ob := orderbook.New()
		if err := ob.LoadSnapshot(snapshot); err != nil {
			log.Printf("[%s] Failed to load snapshot: %v", name, err)
			_ = ex.Close()
			if !wait(backoff, done, interrupt) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		updatesDone := make(chan struct{})
		go func() {
			defer close(updatesDone)
			for update := range ex.Updates() {
				ob.HandleDepthUpdate(update)
			}
		}()

		ob.ProcessBufferedEvents()
		if ob.NeedsResync() {
			// The snapshot and the buffered stream do not line up. Reconnecting
			// is the only reliable way to get a fresh snapshot for adapters
			// whose snapshots come from the WebSocket stream itself.
			log.Printf("[%s] Buffered updates have a sequence gap; reconnecting for a fresh snapshot", name)
			_ = ex.Close()
			if !wait(backoff, done, interrupt) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		books.Set(string(name), ob)
		log.Printf("[%s] Orderbook initialized", name)
		backoff = time.Second

		resync := make(chan struct{})
		go func() {
			ticker := time.NewTicker(reinitCheckInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if ob.NeedsResync() {
						log.Printf("[%s] Orderbook desynchronized; reconnecting for a fresh snapshot", name)
						close(resync)
						return
					}
				case <-updatesDone:
					return
				case <-done:
					return
				case <-interrupt:
					return
				}
			}
		}()

		select {
		case <-updatesDone:
			log.Printf("[%s] Stream closed; reconnecting", name)
		case <-resync:
			log.Printf("[%s] Reconnecting to resynchronize orderbook", name)
		case <-done:
		case <-interrupt:
		}

		books.Delete(string(name))
		_ = ex.Close()

		select {
		case <-done:
			return
		case <-interrupt:
			return
		default:
		}

		if !wait(backoff, done, interrupt) {
			return
		}
		backoff = nextBackoff(backoff)
	}
}

func wait(d time.Duration, done <-chan struct{}, interrupt <-chan os.Signal) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-done:
		return false
	case <-interrupt:
		return false
	}
}

func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > maxReconnectBackoff {
		return maxReconnectBackoff
	}
	return next
}

func printCombinedStats(books *registry.Registry) {
	names := books.Names()
	if len(names) == 0 {
		return
	}

	fmt.Println()

	for i, name := range names {
		ob, ok := books.Get(name)
		if !ok || !ob.IsInitialized() {
			continue
		}

		stats := ob.GetStats()

		midPrice := decimal.Zero
		if !stats.BestBid.IsZero() && !stats.BestAsk.IsZero() {
			midPrice = stats.BestBid.Add(stats.BestAsk).Div(decimal.NewFromInt(2))
		}

		fmt.Printf("%s%s%s", colorBold, name, colorReset)
		fmt.Printf("  Mid: %s%10s%s │ Spread: %s%8s%s | BB: %s%10s%s │ BA: %s%10s%s\n",
			colorYellow, midPrice.StringFixed(2), colorReset,
			colorMagenta, stats.Spread.StringFixed(4), colorReset,
			colorGreen, stats.BestBid.StringFixed(2), colorReset,
			colorRed, stats.BestAsk.StringFixed(2), colorReset)

		fmt.Printf("  DEPTH 0.5%% Bids: %s%9s%s │ Asks: %s%9s%s │ Δ: %s%10s%s\n",
			colorGreen, stats.BidLiquidity05Pct.StringFixed(2), colorReset,
			colorRed, stats.AskLiquidity05Pct.StringFixed(2), colorReset,
			getDeltaColor(stats.DeltaLiquidity05Pct), stats.DeltaLiquidity05Pct.StringFixed(2), colorReset)

		fmt.Printf("  DEPTH 2%%:  Bids: %s%9s%s │ Asks: %s%9s%s │ Δ: %s%10s%s\n",
			colorGreen, stats.BidLiquidity2Pct.StringFixed(2), colorReset,
			colorRed, stats.AskLiquidity2Pct.StringFixed(2), colorReset,
			getDeltaColor(stats.DeltaLiquidity2Pct), stats.DeltaLiquidity2Pct.StringFixed(2), colorReset)

		fmt.Printf("  DEPTH 10%%  Bids: %s%9s%s │ Asks: %s%9s%s │ Δ: %s%10s%s\n",
			colorGreen, stats.BidLiquidity10Pct.StringFixed(2), colorReset,
			colorRed, stats.AskLiquidity10Pct.StringFixed(2), colorReset,
			getDeltaColor(stats.DeltaLiquidity10Pct), stats.DeltaLiquidity10Pct.StringFixed(2), colorReset)

		fmt.Printf("  TOTAL QTY: Bids: %s%9s%s │ Asks: %s%9s%s\n",
			colorGreen, stats.TotalBidsQty.StringFixed(2), colorReset,
			colorRed, stats.TotalAsksQty.StringFixed(2), colorReset)

		if stats.SequenceGaps > 0 || stats.InvalidLevels > 0 {
			fmt.Printf("  HEALTH:   sequence gaps: %d │ invalid levels: %d\n",
				stats.SequenceGaps, stats.InvalidLevels)
		}

		if i < len(names)-1 {
			fmt.Println()
		}
	}
}

func getDeltaColor(delta decimal.Decimal) string {
	if delta.GreaterThan(decimal.Zero) {
		return colorGreen
	} else if delta.LessThan(decimal.Zero) {
		return colorRed
	}
	return colorYellow
}
