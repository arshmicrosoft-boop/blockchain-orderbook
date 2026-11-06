import { memo } from 'react';
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from 'recharts';
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
  type ChartConfig,
} from '@/components/ui/chart';
import { ExchangeBadge } from '@/components/ExchangeBadge';
import type { ChartDataPoint } from '@/types';

type LiquidityChartProps = {
  title: string;
  data: ChartDataPoint[];
  config: ChartConfig;
};

type ExchangeTickProps = {
  x?: number;
  y?: number;
  payload?: { value?: string };
};

// Custom tick component to render exchange badges. Defined outside the chart
// so Recharts receives a stable component reference across renders.
function ExchangeTick({ x = 0, y = 0, payload }: ExchangeTickProps) {
  const value = payload?.value ?? '';
  if (!value) {
    return null;
  }
  const isPerps = value.endsWith('f');

  return (
    <g transform={`translate(${x},${y})`}>
      <foreignObject x={-25} y={0} width={50} height={60}>
        <div className="flex flex-col items-center justify-center gap-1">
          <ExchangeBadge
            exchange={value}
            showLabel={false}
            showMarketType={false}
            iconClassName="w-6 h-6"
          />
          <span className={`text-[8px] font-semibold px-1.5 py-0.5 rounded ${isPerps ? 'bg-primary/20 text-primary' : 'bg-secondary/50 text-secondary-foreground'
            }`}>
            {isPerps ? 'Perps' : 'Spot'}
          </span>
        </div>
      </foreignObject>
    </g>
  );
}

/**
 * Reusable chart component for liquidity visualization
 * Memoized to prevent unnecessary re-renders
 */
export const LiquidityChart = memo(function LiquidityChart({
  title,
  data,
  config,
}: LiquidityChartProps) {
  if (data.length === 0) {
    return null;
  }

  return (
    <div className="rounded-lg border border-border bg-card shadow-sm p-4">
      <h3 className="text-xs font-semibold text-muted-foreground uppercase mb-3">
        {title}
      </h3>
      <ChartContainer config={config} className="min-h-[250px] w-full">
        <BarChart accessibilityLayer data={data}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" />
          <XAxis
            dataKey="exchange"
            tickLine={false}
            tickMargin={5}
            axisLine={false}
            height={70}
            tick={<ExchangeTick />}
          />
          <YAxis
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            className="text-[10px]"
            tickFormatter={(value) => `${value.toLocaleString()}`}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                labelFormatter={(value) => `${value}`}
                formatter={(value) => [`${Number(value).toLocaleString()}`, '']}
              />
            }
          />
          <ChartLegend content={<ChartLegendContent />} />
          <Bar dataKey="bid" fill="var(--color-bid)" radius={[4, 4, 0, 0]} />
          <Bar dataKey="ask" fill="var(--color-ask)" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ChartContainer>
    </div>
  );
});