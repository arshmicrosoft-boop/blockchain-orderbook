import { useEffect, useRef, useState } from 'react';
import type {
  WebSocketMessage,
  OrderbookData,
  StatsData,
} from '@/types';

const RECONNECT_BASE_DELAY = 1000;
const RECONNECT_MAX_DELAY = 30000;

export function useWebSocket(url: string) {
  const [orderbooks, setOrderbooks] = useState<OrderbookData>({});
  const [stats, setStats] = useState<StatsData>({});
  const [isConnected, setIsConnected] = useState(false);
  const [currentSymbol, setCurrentSymbol] = useState('BTCUSDT');
  const [isSwitchingSymbol, setIsSwitchingSymbol] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const switchTimeoutRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    let disposed = false;
    let reconnectDelay = RECONNECT_BASE_DELAY;
    let reconnectTimeout: number | undefined;

    const scheduleReconnect = () => {
      if (disposed) return;
      reconnectTimeout = window.setTimeout(connect, reconnectDelay);
      reconnectDelay = Math.min(reconnectDelay * 2, RECONNECT_MAX_DELAY);
    };

    function connect() {
      if (disposed) return;

      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        if (disposed) return;
        reconnectDelay = RECONNECT_BASE_DELAY;
        setIsConnected(true);
        console.log('WebSocket connected');
      };

      ws.onmessage = (event) => {
        if (disposed) return;

        let message: WebSocketMessage;
        try {
          message = JSON.parse(event.data);
        } catch {
          console.warn('Ignoring malformed WebSocket message');
          return;
        }

        if (message.type === 'orderbook') {
          if (!Array.isArray(message.bids) || !Array.isArray(message.asks)) {
            return;
          }
          setOrderbooks((prev) => ({
            ...prev,
            [message.exchange]: {
              bids: message.bids,
              asks: message.asks,
            },
          }));
        } else if (message.type === 'stats') {
          setStats((prev) => ({
            ...prev,
            [message.exchange]: {
              bestBid: message.bestBid,
              bestAsk: message.bestAsk,
              midPrice: message.midPrice,
              spread: message.spread,
              bidLiquidity05Pct: message.bidLiquidity05Pct,
              askLiquidity05Pct: message.askLiquidity05Pct,
              deltaLiquidity05Pct: message.deltaLiquidity05Pct,
              bidLiquidity2Pct: message.bidLiquidity2Pct,
              askLiquidity2Pct: message.askLiquidity2Pct,
              deltaLiquidity2Pct: message.deltaLiquidity2Pct,
              bidLiquidity10Pct: message.bidLiquidity10Pct,
              askLiquidity10Pct: message.askLiquidity10Pct,
              deltaLiquidity10Pct: message.deltaLiquidity10Pct,
              totalBidsQty: message.totalBidsQty,
              totalAsksQty: message.totalAsksQty,
              totalDelta: message.totalDelta,
            },
          }));
        }
      };

      ws.onerror = (error) => {
        if (disposed) return;
        console.error('WebSocket error:', error);
      };

      ws.onclose = () => {
        if (disposed) return;
        setIsConnected(false);
        if (wsRef.current === ws) {
          wsRef.current = null;
        }
        console.log(`WebSocket disconnected, reconnecting in ${reconnectDelay}ms...`);
        scheduleReconnect();
      };
    }

    connect();

    return () => {
      disposed = true;
      if (reconnectTimeout !== undefined) {
        clearTimeout(reconnectTimeout);
      }
      if (switchTimeoutRef.current !== undefined) {
        clearTimeout(switchTimeoutRef.current);
      }

      const ws = wsRef.current;
      wsRef.current = null;
      if (ws) {
        // Detach handlers so closing during teardown cannot schedule a reconnect.
        ws.onopen = null;
        ws.onmessage = null;
        ws.onerror = null;
        ws.onclose = null;
        ws.close();
      }
    };
  }, [url]);

  const setTickLevel = (tick: number) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'set_tick', tick }));
    }
  };

  const setSymbol = (symbol: string) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      setIsSwitchingSymbol(true);
      setOrderbooks({});
      setStats({});
      setCurrentSymbol(symbol);
      wsRef.current.send(JSON.stringify({ type: 'change_symbol', symbol }));

      if (switchTimeoutRef.current !== undefined) {
        clearTimeout(switchTimeoutRef.current);
      }
      switchTimeoutRef.current = window.setTimeout(() => {
        switchTimeoutRef.current = undefined;
        setIsSwitchingSymbol(false);
      }, 3000);
    }
  };

  return { orderbooks, stats, isConnected, currentSymbol, isSwitchingSymbol, setTickLevel, setSymbol };
}
