import { accountTypeLabels, type NetWorthRow } from '../api'
import { formatCents } from '../money'
import { accountTypeSlot } from './tokens'
import { tooltipStyle, useECharts } from './useECharts'

/**
 * Stacked area of net worth by account type, monthly carry-forward. Account
 * types keep fixed palette slots (identity, not rank).
 */
export function NetWorthArea({ rows }: { rows: NetWorthRow[] }) {
  const ref = useECharts(
    (t) => {
      const months = [...new Set(rows.map((r) => r.month))].sort()
      const types = [...new Set(rows.map((r) => r.type))].sort(
        (a, b) => (accountTypeSlot[a] ?? 9) - (accountTypeSlot[b] ?? 9),
      )
      const byType = new Map(types.map((ty) => [ty, months.map(() => 0)]))
      for (const r of rows) {
        byType.get(r.type)![months.indexOf(r.month)] = r.totalCents / 100
      }

      return {
        tooltip: {
          trigger: 'axis',
          ...tooltipStyle(t),
          valueFormatter: (v: number) => formatCents(Math.round((v as number) * 100)),
        },
        legend: {
          bottom: 0,
          textStyle: { color: t.textSecondary, fontSize: 11 },
          itemWidth: 10,
          itemHeight: 10,
        },
        grid: { left: 72, right: 16, top: 16, bottom: 40 },
        xAxis: {
          type: 'category',
          boundaryGap: false,
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
        series: types.map((ty) => {
          const color = t.series[accountTypeSlot[ty] ?? 7]
          return {
            name: accountTypeLabels[ty as keyof typeof accountTypeLabels] ?? ty,
            type: 'line',
            stack: 'net-worth',
            areaStyle: { opacity: 0.35, color },
            lineStyle: { width: 2, color },
            itemStyle: { color },
            symbol: 'circle',
            symbolSize: 6,
            showSymbol: months.length <= 18,
            data: byType.get(ty),
          }
        }),
      }
    },
    [rows],
  )
  return <div ref={ref} className="h-96 w-full" />
}
