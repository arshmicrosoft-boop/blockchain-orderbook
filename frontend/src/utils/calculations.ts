import type { StatsData, MarketFilter } from '@/types';

/**
 * Filters exchanges based on market type
 */
export function filterExchangesByMarket(
  exchange: string,
  filter: MarketFilter
): boolean {
  if (filter === 'all') return true;
  if (filter === 'spot') return !exchange.endsWith('f');
  if (filter === 'perps') return exchange.endsWith('f');
  return true;
}

/**
 * Calculates reference mid price from multiple exchanges
 */
export function calculateReferenceMidPrice(
  stats: StatsData,
  exchanges: string[]
): number {
  const midPrices = exchanges
    .map((exchange) => stats[exchange])
    .filter(
      (stat) =>
        stat && !isNaN(parseFloat(stat.midPrice)) && parseFloat(stat.midPrice) > 0
    )
    .map((stat) => parseFloat(stat.midPrice));

  return midPrices.length > 0
    ? midPrices.reduce((sum, price) => sum + price, 0) / midPrices.length
    : 0;
}

/**
 * Calculates maximum cumulative value for depth visualization
 */
export function calculateMaxCumulative(
  levels: Array<{ cumulative: string }>
): number {
  if (levels.length === 0) return 1;
  return Math.max(1, ...levels.map((level) => parseFloat(level.cumulative)));
}

/**
 * Formats a number for display with appropriate precision.
 * Non-finite values (missing/N/A fields) render as zero instead of "NaN".
 */
export function formatNumber(value: number, decimals: number = 2): string {
  return (Number.isFinite(value) ? value : 0).toFixed(decimals);
}

/**
 * Formats a large number with locale-specific separators
 */
export function formatLargeNumber(value: number): string {
  return value.toLocaleString();
}

/**
 * Extracts the base exchange name (without 'f' suffix for futures)
 */
export function getBaseExchangeName(exchange: string): string {
  return exchange.endsWith('f') ? exchange.slice(0, -1) : exchange;
}

/**
 * Checks if an exchange is a futures/perps market
 */
export function isFuturesExchange(exchange: string): boolean {
  return exchange.endsWith('f');
}

/**
 * Sorts exchanges so that spot and perps from the same exchange appear together
 * Spot markets appear before their corresponding perps
 */
export function sortExchangesByGroup<T>(exchanges: [string, T][]): [string, T][] {
  return exchanges.sort((a, b) => {
    const [exchangeA] = a;
    const [exchangeB] = b;

    const baseA = getBaseExchangeName(exchangeA);
    const baseB = getBaseExchangeName(exchangeB);

    // First, sort by base exchange name
    if (baseA !== baseB) {
      return baseA.localeCompare(baseB);
    }

    // If same base exchange, spot comes before futures
    const isAFutures = isFuturesExchange(exchangeA);
    const isBFutures = isFuturesExchange(exchangeB);

    if (isAFutures && !isBFutures) return 1;
    if (!isAFutures && isBFutures) return -1;

    return 0;
  });
}