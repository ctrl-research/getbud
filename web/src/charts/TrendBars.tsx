import type { TrendRow } from '../api'
import { formatCents } from '../money'
import { tooltipStyle, useECharts } from './useECharts'

const MAX_SERIES = 8

/**
 * Monthly stacked bars by category. Categories keep their identity color;
 * beyond the top N by total, the tail folds into "Other".
 */
export function TrendBars({ rows, kind }: { rows: TrendRow[]; kind: 'expense' | 'income' }) {
  const ref = useECharts(
    (t) => {
      const months = [...new Set(rows.map((r) => r.month))].sort()
      const value = (r: TrendRow) => (kind === 'expense' ? r.outflowCents : r.inflowCents)

      const totals = new Map<string, { color: string; total: number }>()
      for (const r of rows) {
        if (r.kind !== kind && !(r.kind === '' && value(r) > 0)) continue
        const name = r.kind === '' ? 'Uncategorized' : r.name
        const cur = totals.get(name) ?? { color: r.color, total: 0 }
        cur.total += value(r)
        totals.set(name, cur)
      }
      const ranked = [...totals.entries()].sort((a, b) => b[1].total - a[1].total)
      const keep = new Set(ranked.slice(0, MAX_SERIES - 1).map(([name]) => name))
      const foldOther = ranked.length > MAX_SERIES

      const byName = new Map<string, { color: string; values: number[] }>()
      for (const r of rows) {
        if (r.kind !== kind && !(r.kind === '' && value(r) > 0)) continue
        let name = r.kind === '' ? 'Uncategorized' : r.name
        let color = r.color
        if (foldOther && !keep.has(name)) {
          name = 'Other'
          color = ''
        }
        let s = byName.get(name)
        if (!s) {
          s = { color, values: months.map(() => 0) }
          byName.set(name, s)
        }
        s.values[months.indexOf(r.month)] += value(r) / 100
      }

      // Series ordered by total, biggest at the bottom of the stack.
      const ordered = [...byName.entries()].sort(
        (a, b) => b[1].values.reduce((x, y) => x + y, 0) - a[1].values.reduce((x, y) => x + y, 0),
      )

      return {
        tooltip: {
          trigger: 'axis',
          axisPointer: { type: 'shadow' },
          ...tooltipStyle(t),
          valueFormatter: (v: number) => formatCents(Math.round((v as number) * 100)),
        },
        legend: {
          type: 'scroll',
          bottom: 0,
          textStyle: { color: t.textSecondary, fontSize: 11 },
          itemWidth: 10,
          itemHeight: 10,
        },
        grid: { left: 64, right: 16, top: 16, bottom: 40 },
        xAxis: {
          type: 'category',
          data: months.map((m) => m.slice(0, 7)),
          axisLine: { lineStyle: { color: t.baseline } },
          axisLabel: { color: t.muted, fontSize: 11 },
          axisTick: { show: false },
        },
        yAxis: {
          type: 'value',
          splitLine: { lineStyle: { color: t.gridline } },
          axisLabel: { color: t.muted, fontSize: 11, formatter: (v: number) => `$${v.toLocaleString('en-CA')}` },
        },
        series: ordered.map(([name, s]) => ({
          name,
          type: 'bar',
          stack: 'total',
          barMaxWidth: 36,
          data: s.values,
          itemStyle: {
            color: s.color || t.muted,
            // 2px surface gap between stacked segments.
            borderColor: t.surface,
            borderWidth: 1,
          },
        })),
      }
    },
    [rows, kind],
  )
  return <div ref={ref} className="h-96 w-full" />
}
