import type { CategoryFlow } from '../api'
import { formatCents } from '../money'
import { tooltipStyle, useECharts } from './useECharts'

/** Expense breakdown donut; slices keep category identity colors. */
export function CategoryDonut({ flows }: { flows: CategoryFlow[] }) {
  const ref = useECharts(
    (t) => ({
      tooltip: {
        trigger: 'item',
        ...tooltipStyle(t),
        formatter: (p: { name: string; value: number; percent: number }) =>
          `${p.name}<br/><b>${formatCents(Math.round(p.value * 100))}</b> · ${p.percent}%`,
      },
      series: [
        {
          type: 'pie',
          radius: ['45%', '75%'],
          padAngle: 1,
          data: flows.map((f) => ({
            name: f.name,
            value: f.amountCents / 100,
            itemStyle: { color: f.color || t.muted, borderColor: t.surface, borderWidth: 2 },
          })),
          label: { color: t.textSecondary, fontSize: 12 },
          labelLine: { lineStyle: { color: t.baseline } },
        },
      ],
    }),
    [flows],
  )
  return <div ref={ref} className="h-96 w-full" />
}
