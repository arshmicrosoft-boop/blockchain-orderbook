import { useMemo, useState } from 'react'
import {
  type ColumnDef,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ExchangeBadge } from '@/components/ExchangeBadge'
import type { StatsData, MarketFilter } from '@/types'
import { filterExchangesByMarket, sortExchangesByGroup } from '@/utils/calculations'

type ExchangeStats = {
  exchange: string
  bestBid: string
  bestAsk: string
  midPrice: string
  spread: string
  bidLiquidity05Pct: string
  askLiquidity05Pct: string
  deltaLiquidity05Pct: string
  bidLiquidity2Pct: string
  askLiquidity2Pct: string
  deltaLiquidity2Pct: string
  bidLiquidity10Pct: string
  askLiquidity10Pct: string
  deltaLiquidity10Pct: string
  totalBidsQty: string
  totalAsksQty: string
  totalDelta: string
}

const createColumns = (): ColumnDef<ExchangeStats>[] => [
  {
    accessorKey: 'exchange',
    header: 'Exchange',
    cell: ({ row }) => (
      <ExchangeBadge
        exchange={row.getValue('exchange')}
        showLabel={false}
        showMarketType={true}
        iconClassName="w-6 h-6"
      />
    ),
    sortingFn: 'alphanumeric',
  },
  {
    accessorKey: 'midPrice',
    header: 'Mid Price',
    cell: ({ row }) => (
      <div className="font-mono">{parseFloat(row.getValue('midPrice')).toFixed(2)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('midPrice'))
      const b = parseFloat(rowB.getValue('midPrice'))
      return a - b
    },
  },
  {
    accessorKey: 'bestBid',
    header: 'Best Bid',
    cell: ({ row }) => (
      <div className="font-mono text-green-500">{parseFloat(row.getValue('bestBid')).toFixed(2)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('bestBid'))
      const b = parseFloat(rowB.getValue('bestBid'))
      return a - b
    },
  },
  {
    accessorKey: 'bestAsk',
    header: 'Best Ask',
    cell: ({ row }) => (
      <div className="font-mono text-red-500">{parseFloat(row.getValue('bestAsk')).toFixed(2)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('bestAsk'))
      const b = parseFloat(rowB.getValue('bestAsk'))
      return a - b
    },
  },
  {
    accessorKey: 'bidLiquidity05Pct',
    header: 'Bid Qty 0.5%',
    cell: ({ row }) => (
      <div className="font-mono text-xs">{parseFloat(row.getValue('bidLiquidity05Pct')).toFixed(0)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('bidLiquidity05Pct'))
      const b = parseFloat(rowB.getValue('bidLiquidity05Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'askLiquidity05Pct',
    header: 'Ask Qty 0.5%',
    cell: ({ row }) => (
      <div className="font-mono text-xs">{parseFloat(row.getValue('askLiquidity05Pct')).toFixed(0)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('askLiquidity05Pct'))
      const b = parseFloat(rowB.getValue('askLiquidity05Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'deltaLiquidity05Pct',
    header: 'Δ Liq 0.5%',
    cell: ({ row }) => {
      const value = parseFloat(row.getValue('deltaLiquidity05Pct'))
      return (
        <div className={`font-mono text-xs ${value > 0 ? 'text-green-500' : value < 0 ? 'text-red-500' : 'text-yellow-500'}`}>
          {value.toFixed(0)}
        </div>
      )
    },
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('deltaLiquidity05Pct'))
      const b = parseFloat(rowB.getValue('deltaLiquidity05Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'bidLiquidity2Pct',
    header: 'Bid Qty 2%',
    cell: ({ row }) => (
      <div className="font-mono text-xs">{parseFloat(row.getValue('bidLiquidity2Pct')).toFixed(0)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('bidLiquidity2Pct'))
      const b = parseFloat(rowB.getValue('bidLiquidity2Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'askLiquidity2Pct',
    header: 'Ask Qty 2%',
    cell: ({ row }) => (
      <div className="font-mono text-xs">{parseFloat(row.getValue('askLiquidity2Pct')).toFixed(0)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('askLiquidity2Pct'))
      const b = parseFloat(rowB.getValue('askLiquidity2Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'deltaLiquidity2Pct',
    header: 'Δ Liq 2%',
    cell: ({ row }) => {
      const value = parseFloat(row.getValue('deltaLiquidity2Pct'))
      return (
        <div className={`font-mono text-xs ${value > 0 ? 'text-green-500' : value < 0 ? 'text-red-500' : 'text-yellow-500'}`}>
          {value.toFixed(0)}
        </div>
      )
    },
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('deltaLiquidity2Pct'))
      const b = parseFloat(rowB.getValue('deltaLiquidity2Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'bidLiquidity10Pct',
    header: 'Bid Qty 10%',
    cell: ({ row }) => (
      <div className="font-mono text-xs">{parseFloat(row.getValue('bidLiquidity10Pct')).toFixed(0)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('bidLiquidity10Pct'))
      const b = parseFloat(rowB.getValue('bidLiquidity10Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'askLiquidity10Pct',
    header: 'Ask Qty 10%',
    cell: ({ row }) => (
      <div className="font-mono text-xs">{parseFloat(row.getValue('askLiquidity10Pct')).toFixed(0)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('askLiquidity10Pct'))
      const b = parseFloat(rowB.getValue('askLiquidity10Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'deltaLiquidity10Pct',
    header: 'Δ Liq 10%',
    cell: ({ row }) => {
      const value = parseFloat(row.getValue('deltaLiquidity10Pct'))
      return (
        <div className={`font-mono text-xs ${value > 0 ? 'text-green-500' : value < 0 ? 'text-red-500' : 'text-yellow-500'}`}>
          {value.toFixed(0)}
        </div>
      )
    },
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('deltaLiquidity10Pct'))
      const b = parseFloat(rowB.getValue('deltaLiquidity10Pct'))
      return a - b
    },
  },
  {
    accessorKey: 'totalBidsQty',
    header: 'Total Bids Qty',
    cell: ({ row }) => (
      <div className="font-mono text-xs text-green-500">{parseFloat(row.getValue('totalBidsQty')).toFixed(0)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('totalBidsQty'))
      const b = parseFloat(rowB.getValue('totalBidsQty'))
      return a - b
    },
  },
  {
    accessorKey: 'totalAsksQty',
    header: 'Total Asks Qty',
    cell: ({ row }) => (
      <div className="font-mono text-xs text-red-500">{parseFloat(row.getValue('totalAsksQty')).toFixed(0)}</div>
    ),
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('totalAsksQty'))
      const b = parseFloat(rowB.getValue('totalAsksQty'))
      return a - b
    },
  },
  {
    accessorKey: 'totalDelta',
    header: 'Total Δ',
    cell: ({ row }) => {
      const value = parseFloat(row.getValue('totalDelta'))
      return (
        <div className={`font-mono text-xs font-semibold ${value > 0 ? 'text-green-500' : value < 0 ? 'text-red-500' : 'text-yellow-500'}`}>
          {value.toFixed(0)}
        </div>
      )
    },
    sortingFn: (rowA, rowB) => {
      const a = parseFloat(rowA.getValue('totalDelta'))
      const b = parseFloat(rowB.getValue('totalDelta'))
      return a - b
    },
  },
]

type StatsTableProps = {
  stats: StatsData
  filter?: MarketFilter
}

export function StatsTable({
  stats,
  filter = 'all',
}: StatsTableProps) {
  const [sorting, setSorting] = useState<SortingState>([])

  const data = useMemo(() => {
    const filtered = Object.entries(stats)
      .filter(([exchange]) => filterExchangesByMarket(exchange, filter));

    const sorted = sortExchangesByGroup(filtered);

    return sorted.map(([exchange, stat]) => ({
      exchange,
      ...stat,
    }));
  }, [stats, filter])

  const columns = useMemo(() => createColumns(), [])

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
    },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getRowId: (row) => row.exchange,
  })

  if (data.length === 0) {
    return null
  }

  return (
    <div className="rounded-lg border border-border bg-card shadow-sm">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead
                  key={header.id}
                  className="text-muted-foreground text-xs font-medium"
                >
                  {header.isPlaceholder ? null : (
                    <div
                      className={`flex items-center gap-1 ${header.column.getCanSort() ? 'cursor-pointer select-none' : ''
                        }`}
                      onClick={header.column.getToggleSortingHandler()}
                    >
                      {flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                      {{
                        asc: ' ↑',
                        desc: ' ↓',
                      }[header.column.getIsSorted() as string] ?? null}
                    </div>
                  )}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows?.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow key={row.id} className="hover:bg-muted/40">
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                No data
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
