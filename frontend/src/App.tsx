import { useMemo, useState } from 'react';
import { useLocalStorage } from '@uidotdev/usehooks';
import { useWebSocket } from './hooks/useWebSocket';
import { useTheme } from './hooks/useTheme';
import { useChartData } from './hooks/useChartData';
import { useAggregatedOrderbook } from './hooks/useAggregatedOrderbook';
import { StatsTable } from './components/StatsTable';
import { OrderbookCard } from './components/OrderbookCard';
import { LiquidityChart } from './components/LiquidityChart';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './components/ui/select';
import { Button } from './components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from './components/ui/tooltip';
import { ToggleGroup, ToggleGroupItem } from './components/ui/toggle-group';
import { Moon, Sun, Layers } from 'lucide-react';
import { TICK_LEVELS, CHART_CONFIG, POPULAR_SYMBOLS } from './constants';
import { filterExchangesByMarket, sortExchangesByGroup } from './utils/calculations';
import type { MarketFilter } from './types';
import bingxLogo from '@/assets/bingx.png';

// The backend defaults to port 8086. VITE_WS_URL can override the full URL when
// the frontend and backend are served from different hosts.
const WS_URL =
  import.meta.env.VITE_WS_URL ??
  `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${
    window.location.hostname || 'localhost'
  }:8086/ws`;

function App() {
  const { isDark, toggleTheme } = useTheme();
  const [marketFilter, setMarketFilter] = useLocalStorage<MarketFilter>('marketFilter', 'all');
  const [showAggregate, setShowAggregate] = useState(false);
  const { orderbooks, stats, isConnected, currentSymbol, isSwitchingSymbol, setTickLevel, setSymbol } = useWebSocket(WS_URL);
  const { chartData05Pct, chartData2Pct, chartData10Pct, chartDataTotal } = useChartData(stats, marketFilter);

  // Filter and sort orderbooks based on market filter
  const filteredOrderbooks = useMemo(() => {
    const filtered = Object.entries(orderbooks).filter(([exchange]) =>
      filterExchangesByMarket(exchange, marketFilter)
    );
    return sortExchangesByGroup(filtered);
  }, [orderbooks, marketFilter]);

  // Get aggregated orderbook for filtered exchanges
  const filteredOrderbooksData = useMemo(() => {
    return Object.fromEntries(filteredOrderbooks);
  }, [filteredOrderbooks]);

  const aggregated = useAggregatedOrderbook(filteredOrderbooksData);

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="sticky top-0 z-50 bg-card border-b border-border backdrop-blur-sm">
        <div className="max-w-[1400px] mx-auto px-4 md:px-6 lg:px-8 py-2.5">
          <a
            href="https://bingx.com/en/invite/JGNQPF"
            target="_blank"
            rel="noopener noreferrer"
            className="flex flex-col sm:flex-row items-center justify-center gap-1 sm:gap-2 text-sm hover:opacity-80 transition-opacity"
          >
            <span className="text-muted-foreground text-center sm:text-left">
              If you'd like to support my work, consider signing up to BingX
            </span>
            <img src={bingxLogo} alt="BingX" className="h-5" />
            <span className="text-primary font-medium">
              with my referral
            </span>
            <span className="hidden sm:inline text-muted-foreground">•</span>
            <span className="text-xs text-muted-foreground">
              Low fees, great UX
            </span>
          </a>
        </div>
      </div>
      <div className="max-w-[1400px] mx-auto px-4 md:px-6 lg:px-8 py-6">
        <div className="mb-6 flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold tracking-tight">Crypto Orderbook</h1>
            <Select value={currentSymbol} onValueChange={setSymbol} disabled={isSwitchingSymbol}>
              <SelectTrigger className="w-[140px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {POPULAR_SYMBOLS.map((symbol) => (
                  <SelectItem key={symbol.value} value={symbol.value}>
                    {symbol.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <div className="hidden sm:inline-flex items-center gap-2 rounded-full border border-border bg-muted/30 px-2 py-1">
              <span
                className={`size-2 rounded-full ${isConnected && !isSwitchingSymbol ? 'bg-green-500' : isSwitchingSymbol ? 'bg-yellow-500' : 'bg-destructive'
                  }`}
              />
              <span className="text-xs text-muted-foreground">
                {isSwitchingSymbol ? 'Switching...' : isConnected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <ToggleGroup
              type="single"
              value={marketFilter}
              onValueChange={(value) => value && setMarketFilter(value as MarketFilter)}
              variant="outline"
            >
              <ToggleGroupItem value="all" aria-label="Show all markets" className="px-2.5">
                All
              </ToggleGroupItem>
              <ToggleGroupItem value="spot" aria-label="Show spot markets" className="px-2.5">
                Spot
              </ToggleGroupItem>
              <ToggleGroupItem value="perps" aria-label="Show perpetual futures" className="px-2.5">
                Perps
              </ToggleGroupItem>
            </ToggleGroup>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  aria-label="Toggle theme"
                  onClick={toggleTheme}
                >
                  {isDark ? <Sun className="size-4" /> : <Moon className="size-4" />}
                </Button>
              </TooltipTrigger>
              <TooltipContent>Toggle theme</TooltipContent>
            </Tooltip>
          </div>
        </div>

        <div className="space-y-6">
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-semibold tracking-wide text-muted-foreground uppercase">
                Exchange Statistics
              </h2>
            </div>
            <StatsTable stats={stats} filter={marketFilter} />
          </section>

          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-semibold tracking-wide text-muted-foreground uppercase">
                Order Books
              </h2>
              <div className="flex items-center gap-2">
                <span className="hidden md:block text-xs text-muted-foreground">Tick</span>
                <Select defaultValue="1" onValueChange={(value) => setTickLevel(parseFloat(value))}>
                  <SelectTrigger size="sm" className="w-[84px]">
                    <SelectValue placeholder="Tick" />
                  </SelectTrigger>
                  <SelectContent>
                    {TICK_LEVELS.map((tick) => (
                      <SelectItem key={tick.value} value={tick.value}>
                        {tick.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={showAggregate ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setShowAggregate(!showAggregate)}
                      className="px-2 gap-1.5"
                    >
                      <Layers className="size-3.5" />
                      <span className="text-xs">Aggregate</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {showAggregate ? 'Show individual exchanges' : 'Show aggregated orderbook'}
                  </TooltipContent>
                </Tooltip>
              </div>
            </div>
            <div className="flex gap-4 overflow-x-auto pb-2">
              {showAggregate ? (
                <OrderbookCard
                  exchange="Aggregated"
                  bids={aggregated.bids}
                  asks={aggregated.asks}
                  stats={aggregated.stats}
                />
              ) : (
                filteredOrderbooks.map(([exchange, data]) => (
                  <OrderbookCard
                    key={exchange}
                    exchange={exchange}
                    bids={data.bids}
                    asks={data.asks}
                    stats={stats[exchange]}
                  />
                ))
              )}
            </div>
          </section>

          {chartData05Pct.length > 0 && (
            <section>
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <LiquidityChart
                  title="Liquidity at 0.5% Depth"
                  data={chartData05Pct}
                  config={CHART_CONFIG}
                />
                <LiquidityChart
                  title="Liquidity at 2% Depth"
                  data={chartData2Pct}
                  config={CHART_CONFIG}
                />
                <LiquidityChart
                  title="Liquidity at 10% Depth"
                  data={chartData10Pct}
                  config={CHART_CONFIG}
                />
                <LiquidityChart
                  title="Total Liquidity"
                  data={chartDataTotal}
                  config={CHART_CONFIG}
                />
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
