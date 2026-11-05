import { useMemo } from 'react';
import type { StatsData, MarketFilter } from '@/types';
import { filterExchangesByMarket, sortExchangesByGroup } from '@/utils/calculations';

/**
 * Parse a numeric field, falling back to 0 when the value is missing or
 * non-finite. Recharts renders nothing useful for NaN/Infinity.
 */
function toFiniteNumber(value: string | undefined): number {
  const parsed = parseFloat(value ?? '');
  return Number.isFinite(parsed) ? parsed : 0;
}

/**
 * Hook to transform stats data into chart-ready format
 * Memoized for performance
 */
export function useChartData(stats: StatsData, filter: MarketFilter) {
  const chartData05Pct = useMemo(() => {
    const filtered = Object.entries(stats)
      .filter(([exchange]) => filterExchangesByMarket(exchange, filter));
    const sorted = sortExchangesByGroup(filtered);
    return sorted.map(([exchange, stat]) => ({
      exchange,
      bid: toFiniteNumber(stat.bidLiquidity05Pct),
      ask: toFiniteNumber(stat.askLiquidity05Pct),
    }));
  }, [stats, filter]);

  const chartData2Pct = useMemo(() => {
    const filtered = Object.entries(stats)
      .filter(([exchange]) => filterExchangesByMarket(exchange, filter));
    const sorted = sortExchangesByGroup(filtered);
    return sorted.map(([exchange, stat]) => ({
      exchange,
      bid: toFiniteNumber(stat.bidLiquidity2Pct),
      ask: toFiniteNumber(stat.askLiquidity2Pct),
    }));
  }, [stats, filter]);

  const chartData10Pct = useMemo(() => {
    const filtered = Object.entries(stats)
      .filter(([exchange]) => filterExchangesByMarket(exchange, filter));
    const sorted = sortExchangesByGroup(filtered);
    return sorted.map(([exchange, stat]) => ({
      exchange,
      bid: toFiniteNumber(stat.bidLiquidity10Pct),
      ask: toFiniteNumber(stat.askLiquidity10Pct),
    }));
  }, [stats, filter]);

  const chartDataTotal = useMemo(() => {
    const filtered = Object.entries(stats)
      .filter(([exchange]) => filterExchangesByMarket(exchange, filter));
    const sorted = sortExchangesByGroup(filtered);
    return sorted.map(([exchange, stat]) => ({
      exchange,
      bid: toFiniteNumber(stat.totalBidsQty),
      ask: toFiniteNumber(stat.totalAsksQty),
    }));
  }, [stats, filter]);

  return {
    chartData05Pct,
    chartData2Pct,
    chartData10Pct,
    chartDataTotal,
  };
}
