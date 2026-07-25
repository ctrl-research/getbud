import { useEffect, useRef, useSyncExternalStore } from 'react'
import * as echarts from 'echarts/core'
import { BarChart, LineChart, PieChart, SankeyChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsCoreOption } from 'echarts/core'
import { getTheme, subscribeTheme } from '../theme'
import { currentTokens, type ChartTokens } from './tokens'

echarts.use([BarChart, LineChart, PieChart, SankeyChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer])

/**
 * Mounts an ECharts instance in a div, rebuilding on data or theme change
 * and resizing with its container. `build` receives the mode's color tokens.
 */
export function useECharts(build: (tokens: ChartTokens) => EChartsCoreOption, deps: unknown[]) {
  const ref = useRef<HTMLDivElement>(null)
  const chartRef = useRef<echarts.ECharts | null>(null)
  // Re-render when the theme (or the OS setting under "system") changes.
  const theme = useSyncExternalStore(subscribeTheme, getTheme)

  useEffect(() => {
    if (!ref.current) return
    const chart = echarts.init(ref.current)
    chartRef.current = chart
    const observer = new ResizeObserver(() => chart.resize())
    observer.observe(ref.current)
    return () => {
      observer.disconnect()
      chart.dispose()
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    chartRef.current?.setOption(build(currentTokens()), { notMerge: true })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, theme])

  return ref
}

/** Shared tooltip styling on the mode's surface. */
export function tooltipStyle(t: ChartTokens) {
  return {
    backgroundColor: t.surface,
    borderColor: t.gridline,
    textStyle: { color: t.textPrimary, fontSize: 12 },
  }
}
