import { useMemo } from 'react';
import type { OrderbookData, OrderbookLevel } from '@/types';

type NettedLevel = {
  price: number;
  bidQty: number;
  askQty: number;
};

export function useAggregatedOrderbook(orderbooks: OrderbookData) {
  return useMemo(() => {
    // Key by numeric price so "100.5" and "100.50" from different exchanges
    // aggregate into a single level instead of producing duplicate rows.
    const priceMap = new Map<number, NettedLevel>();

    const addLevel = (
      priceText: string,
      quantityText: string,
      side: 'bid' | 'ask'
    ) => {
      const price = parseFloat(priceText);
      const quantity = parseFloat(quantityText);
      if (!Number.isFinite(price) || !Number.isFinite(quantity) || quantity <= 0) {
        return;
      }

      const existing = priceMap.get(price);
      if (existing) {
        if (side === 'bid') {
          existing.bidQty += quantity;
        } else {
          existing.askQty += quantity;
        }
        return;
      }

      priceMap.set(price, {
        price,
        bidQty: side === 'bid' ? quantity : 0,
        askQty: side === 'ask' ? quantity : 0,
      });
    };

    Object.values(orderbooks).forEach(({ bids, asks }) => {
      bids.forEach((bid) => addLevel(bid.price, bid.quantity, 'bid'));
      asks.forEach((ask) => addLevel(ask.price, ask.quantity, 'ask'));
    });

    const nettedBids: [number, number][] = [];
    const nettedAsks: [number, number][] = [];

    priceMap.forEach(({ price, bidQty, askQty }) => {
      const net = bidQty - askQty;
      if (net > 0) {
        nettedBids.push([price, net]);
      } else if (net < 0) {
        nettedAsks.push([price, Math.abs(net)]);
      }
    });

    nettedBids.sort((a, b) => b[0] - a[0]);
    nettedAsks.sort((a, b) => a[0] - b[0]);

    let bidCumulative = 0;
    const bids: OrderbookLevel[] = nettedBids.map(([price, quantity]) => {
      bidCumulative += quantity;
      return {
        price: price.toString(),
        quantity: quantity.toString(),
        cumulative: bidCumulative.toString(),
      };
    });

    let askCumulative = 0;
    const asks: OrderbookLevel[] = nettedAsks.map(([price, quantity]) => {
      askCumulative += quantity;
      return {
        price: price.toString(),
        quantity: quantity.toString(),
        cumulative: askCumulative.toString(),
      };
    });

    const bestBid = bids.length > 0 ? parseFloat(bids[0].price) : 0;
    const bestAsk = asks.length > 0 ? parseFloat(asks[0].price) : 0;
    const midPrice = bestBid && bestAsk ? (bestBid + bestAsk) / 2 : 0;
    const spread = bestBid && bestAsk ? bestAsk - bestBid : 0;

    // Calculate liquidity at different depth percentages
    const calculateLiquidity = (levels: OrderbookLevel[], referencePrice: number, percentage: number, isBid: boolean) => {
      if (!Number.isFinite(referencePrice) || referencePrice <= 0) {
        return 0;
      }

      const threshold = isBid
        ? referencePrice * (1 - percentage / 100)
        : referencePrice * (1 + percentage / 100);

      let liquidity = 0;
      for (const level of levels) {
        const price = parseFloat(level.price);
        if (isBid ? price >= threshold : price <= threshold) {
          liquidity += parseFloat(level.quantity);
        } else {
          break;
        }
      }
      return liquidity;
    };

    const bidLiquidity05Pct = calculateLiquidity(bids, midPrice, 0.5, true);
    const askLiquidity05Pct = calculateLiquidity(asks, midPrice, 0.5, false);
    const bidLiquidity2Pct = calculateLiquidity(bids, midPrice, 2, true);
    const askLiquidity2Pct = calculateLiquidity(asks, midPrice, 2, false);
    const bidLiquidity10Pct = calculateLiquidity(bids, midPrice, 10, true);
    const askLiquidity10Pct = calculateLiquidity(asks, midPrice, 10, false);

    return {
      bids,
      asks,
      stats: {
        bestBid: bestBid.toString(),
        bestAsk: bestAsk.toString(),
        midPrice: midPrice.toString(),
        spread: spread.toString(),
        bidLiquidity05Pct: bidLiquidity05Pct.toString(),
        askLiquidity05Pct: askLiquidity05Pct.toString(),
        deltaLiquidity05Pct: (bidLiquidity05Pct - askLiquidity05Pct).toString(),
        bidLiquidity2Pct: bidLiquidity2Pct.toString(),
        askLiquidity2Pct: askLiquidity2Pct.toString(),
        deltaLiquidity2Pct: (bidLiquidity2Pct - askLiquidity2Pct).toString(),
        bidLiquidity10Pct: bidLiquidity10Pct.toString(),
        askLiquidity10Pct: askLiquidity10Pct.toString(),
        deltaLiquidity10Pct: (bidLiquidity10Pct - askLiquidity10Pct).toString(),
        totalBidsQty: bidCumulative.toString(),
        totalAsksQty: askCumulative.toString(),
        totalDelta: (bidCumulative - askCumulative).toString(),
      },
    };
  }, [orderbooks]);
}
