import type { SankeyData } from '../api'
import { formatCents } from '../money'
import { tooltipStyle, useECharts } from './useECharts'

/** Income -> pool -> expense flow. Node colors follow category identity. */
export function SankeyChart({ data }: { data: SankeyData }) {
  const ref = useECharts(
    (t) => ({
      tooltip: {
        trigger: 'item',
        ...tooltipStyle(t),
        formatter: (params: { dataType?: string; name?: string; value?: number; data?: { source?: string; target?: string } }) => {
          if (params.dataType === 'edge') {
            return `${params.data?.source} → ${params.data?.target}<br/><b>${formatCents(params.value ?? 0)}</b>`
          }
          return `${params.name}<br/><b>${formatCents(params.value ?? 0)}</b>`
        },
      },
      series: [
        {
          type: 'sankey',
          nodeWidth: 12,
          nodeGap: 14,
          left: 8,
          right: 130,
          top: 12,
          bottom: 12,
          emphasis: { focus: 'adjacency' },
          data: data.nodes.map((n) => ({
            name: n.name,
            itemStyle: {
              color: n.color || t.muted,
              borderColor: t.surface,
              borderWidth: 1,
            },
          })),
          links: data.links.map((l) => ({
            source: l.source,
            target: l.target,
            value: l.valueCents / 100,
          })),
          lineStyle: { color: 'gradient', opacity: 0.25, curveness: 0.5 },
          label: { color: t.textPrimary, fontSize: 12 },
        },
      ],
    }),
    [data],
  )
  return <div ref={ref} className="h-[28rem] w-full" />
}
