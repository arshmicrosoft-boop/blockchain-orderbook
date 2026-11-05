import { memo, useMemo } from 'react';
import { ExchangeBadge } from '@/components/ExchangeBadge';
import type { OrderbookLevel, StatsData } from '@/types';
import { calculateMaxCumulative, formatNumber } from '@/utils/calculations';

type OrderbookCardProps = {
  exchange: string;
  bids: OrderbookLevel[];
  asks: OrderbookLevel[];
  stats: StatsData[string];
  rowsPerSide?: number;
};

/**
 * Renders a single orderbook card with bids/asks
 * Memoized to prevent unnecessary re-renders
 */
export const OrderbookCard = memo(function OrderbookCard({
  exchange,
  bids: allBids,
  asks: allAsks,
  stats,
  rowsPerSide = 10,
}: OrderbookCardProps) {
  // Derive top-of-book from the levels we actually render so the displayed
  // depth can never be filtered out by a stale `stats` message.
  const bookBestBid = allBids.length > 0 ? parseFloat(allBids[0].price) : NaN;
  const bookBestAsk = allAsks.length > 0 ? parseFloat(allAsks[0].price) : NaN;
  const statsBestBid = stats ? parseFloat(stats.bestBid) : NaN;
  const statsBestAsk = stats ? parseFloat(stats.bestAsk) : NaN;
  const statsMid = stats ? parseFloat(stats.midPrice) : NaN;
  const statsSpread = stats ? parseFloat(stats.spread) : NaN;

  const bestBid = Number.isFinite(bookBestBid) ? bookBestBid : statsBestBid;
  const bestAsk = Number.isFinite(bookBestAsk) ? bookBestAsk : statsBestAsk;
  const hasBothSides = Number.isFinite(bestBid) && Number.isFinite(bestAsk);
  const midPrice = hasBothSides ? (bestBid + bestAsk) / 2 : statsMid;
  const spread = hasBothSides ? bestAsk - bestBid : statsSpread;

  // Calculate bid/ask imbalance
  const imbalance = useMemo(() => {
    if (!stats) return { totalBids: 0, totalAsks: 0, bidPercentage: 50 };

    const totalBids = parseFloat(stats.totalBidsQty) || 0;
    const totalAsks = parseFloat(stats.totalAsksQty) || 0;
    const total = totalBids + totalAsks;

    return {
      totalBids,
      totalAsks,
      bidPercentage: total > 0 ? (totalBids / total) * 100 : 50,
    };
  }, [stats]);

  const maxAskCum = calculateMaxCumulative(allAsks);
  const maxBidCum = calculateMaxCumulative(allBids);

  // Filter and slice relevant levels
  const relevantAsks = allAsks.filter((ask) => parseFloat(ask.price) >= bestAsk);
  const displayAsks = relevantAsks.slice(0, rowsPerSide).reverse();

  const relevantBids = allBids.filter((bid) => parseFloat(bid.price) <= bestBid);
  const displayBids = relevantBids.slice(0, rowsPerSide);

  // Pad with nulls if needed
  const paddedAsks = [
    ...Array(Math.max(0, rowsPerSide - displayAsks.length)).fill(null),
    ...displayAsks,
  ];

  const paddedBids = [
    ...displayBids,
    ...Array(Math.max(0, rowsPerSide - displayBids.length)).fill(null),
  ];

  return (
    <div className="relative rounded-lg border border-border bg-card shadow-sm p-2 flex-shrink-0 w-[300px] mt-4">
      <div className="absolute -top-3 left-1/3 -translate-x-1/3 z-10 bg-card px-2 py-1 rounded-md border border-border flex items-center justify-center">
        {exchange === 'Aggregated' ? (
          <span className="text-xs font-semibold text-foreground">Aggregated</span>
        ) : (
          <ExchangeBadge
            exchange={exchange}
            showLabel={false}
            showMarketType={true}
            iconClassName="w-5 h-5"
          />
        )}
      </div>

      <div className="grid grid-cols-3 gap-2 mb-1 px-2 text-[10px] font-semibold text-muted-foreground uppercase">
        <span>Price</span>
        <span className="text-right">Size</span>
        <span className="text-right">Sum</span>
      </div>

      <div className="space-y-px">
        {paddedAsks.map((ask, idx) => {
          if (ask === null) {
            return (
              <div
                key={`ask-padding-${idx}`}
                className="grid grid-cols-3 gap-2 font-mono text-[11px] leading-tight px-2 py-0.5 rounded opacity-0"
              >
                <span>-</span>
                <span className="text-right">-</span>
                <span className="text-right">-</span>
              </div>
            );
          }
          const pct = (parseFloat(ask.cumulative) / maxAskCum) * 100;
          return (
            <div
              key={`ask-${idx}`}
              className="relative grid grid-cols-3 gap-2 font-mono text-[11px] leading-tight px-2 py-0.5 rounded"
            >
              <div
                className="absolute top-0 bottom-0 right-0 rounded bg-red-500/10"
                style={{ width: `${pct}%` }}
              />
              <span className="relative z-10 text-red-400">
                {formatNumber(parseFloat(ask.price), 2)}
              </span>
              <span className="relative z-10 text-right text-muted-foreground">
                {formatNumber(parseFloat(ask.quantity), 4)}
              </span>
              <span className="relative z-10 text-right text-muted-foreground/80">
                {formatNumber(parseFloat(ask.cumulative), 4)}
              </span>
            </div>
          );
        })}
      </div>

      <div className="my-1.5 py-1.5 px-2 rounded border border-border/60 bg-muted text-center">
        <div className="flex items-center justify-center gap-3 text-xs">
          <div>
            <span className="text-[10px] text-muted-foreground">Mid </span>
            <span className="font-semibold text-yellow-400">
              ${formatNumber(midPrice, 2)}
            </span>
          </div>
          <div className="w-px h-3 bg-border" />
          <div>
            <span className="text-[10px] text-muted-foreground">Spread </span>
            <span className="font-semibold text-blue-400">
              ${formatNumber(spread, 2)}
            </span>
          </div>
        </div>
      </div>

      <div className="space-y-px">
        {paddedBids.map((bid, idx) => {
          if (bid === null) {
            return (
              <div
                key={`bid-padding-${idx}`}
                className="grid grid-cols-3 gap-2 font-mono text-[11px] leading-tight px-2 py-0.5 rounded opacity-0"
              >
                <span>-</span>
                <span className="text-right">-</span>
                <span className="text-right">-</span>
              </div>
            );
          }
          const pct = (parseFloat(bid.cumulative) / maxBidCum) * 100;
          return (
            <div
              key={`bid-${idx}`}
              className="relative grid grid-cols-3 gap-2 font-mono text-[11px] leading-tight px-2 py-0.5 rounded"
            >
              <div
                className="absolute top-0 bottom-0 right-0 rounded bg-green-500/10"
                style={{ width: `${pct}%` }}
              />
              <span className="relative z-10 text-green-400">
                {formatNumber(parseFloat(bid.price), 2)}
              </span>
              <span className="relative z-10 text-right text-muted-foreground">
                {formatNumber(parseFloat(bid.quantity), 4)}
              </span>
              <span className="relative z-10 text-right text-muted-foreground/80">
                {formatNumber(parseFloat(bid.cumulative), 4)}
              </span>
            </div>
          );
        })}
      </div>

      <div className="mt-3 px-2">
        <div className="relative h-6 bg-secondary rounded-lg overflow-hidden">
          <div
            className="absolute left-0 top-0 h-full bg-green-500/30 transition-all duration-300"
            style={{ width: `${imbalance.bidPercentage}%` }}
          />

          <div className="absolute left-1/2 top-0 bottom-0 w-px bg-border/50 -translate-x-1/2" />

          <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 flex items-center justify-center">
            <div className={`px-1.5 py-0.5 rounded text-[9px] font-bold ${imbalance.bidPercentage > 50
              ? 'bg-green-500/80 text-white'
              : imbalance.bidPercentage < 50
                ? 'bg-red-500/80 text-white'
                : 'bg-muted/80 text-muted-foreground'
              }`}>
              {imbalance.bidPercentage > 50
                ? `↑ ${formatNumber(imbalance.bidPercentage - 50, 1)}%`
                : imbalance.bidPercentage < 50
                  ? `↓ ${formatNumber(50 - imbalance.bidPercentage, 1)}%`
                  : '='
              }
            </div>
          </div>

          <div className="absolute inset-0 flex items-center justify-between px-2 text-[10px] font-mono font-semibold">
            <span className="text-green-400">
              Bids: {formatNumber(imbalance.totalBids, 2)}
            </span>
            <span className="text-red-400">
              Asks: {formatNumber(imbalance.totalAsks, 2)}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
});