import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  fetchReportNetWorth,
  fetchReportSankey,
  fetchReportSummary,
  fetchReportTrends,
  type DateRange,
} from '../api'
import { DateRangePicker, rangePresets } from '../components/DateRangePicker'
import { EmptyState } from '../components/ui'
import { formatCents } from '../money'
import { SankeyChart } from '../charts/Sankey'
import { TrendBars } from '../charts/TrendBars'
import { NetWorthArea } from '../charts/NetWorthArea'
import { CategoryDonut } from '../charts/CategoryDonut'

const TABS = ['Cash flow', 'Trends', 'Net worth', 'Categories'] as const
type Tab = (typeof TABS)[number]

export function ReportsPage() {
  const [tab, setTab] = useState<Tab>('Cash flow')
  // Default to last month - the most recent complete picture.
  const [preset, setPreset] = useState<string | null>('last-month')
  const [range, setRange] = useState<DateRange>(() => rangePresets[1].range())

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">Reports</h1>
        <DateRangePicker
          value={range}
          preset={preset}
          onChange={(r, p) => {
            setRange(r)
            setPreset(p)
          }}
        />
      </div>

      <div className="mb-6 flex gap-1 border-b border-slate-200 dark:border-slate-800">
        {TABS.map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={`border-b-2 px-4 py-2 text-sm ${
              tab === t
                ? 'border-slate-900 font-medium text-slate-900 dark:border-slate-100 dark:text-slate-100'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === 'Cash flow' && <CashFlowTab range={range} />}
      {tab === 'Trends' && <TrendsTab range={range} />}
      {tab === 'Net worth' && <NetWorthTab range={range} />}
      {tab === 'Categories' && <CategoriesTab range={range} />}
    </div>
  )
}

function Card({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4">
      {children}
    </div>
  )
}

function CashFlowTab({ range }: { range: DateRange }) {
  const { data } = useQuery({ queryKey: ['report-sankey', range], queryFn: () => fetchReportSankey(range) })
  if (!data) return null
  if (data.links.length === 0) {
    return <EmptyState>No income or expenses in this period — add or import some transactions first.</EmptyState>
  }
  return (
    <Card>
      <p className="mb-2 text-sm text-slate-500 dark:text-slate-400">
        Where money came from and where it went. Transfers between your own accounts are excluded.
      </p>
      <SankeyChart data={data} />
    </Card>
  )
}

function TrendsTab({ range }: { range: DateRange }) {
  const [kind, setKind] = useState<'expense' | 'income'>('expense')
  const { data } = useQuery({ queryKey: ['report-trends', range], queryFn: () => fetchReportTrends(range) })
  if (!data) return null
  if (data.rows.length === 0) {
    return <EmptyState>No transactions in this period.</EmptyState>
  }
  return (
    <Card>
      <div className="mb-2 flex items-center justify-between">
        <p className="text-sm text-slate-500 dark:text-slate-400">Monthly {kind}s by category.</p>
        <div className="flex gap-1">
          {(['expense', 'income'] as const).map((k) => (
            <button
              key={k}
              type="button"
              onClick={() => setKind(k)}
              className={`rounded-lg px-3 py-1 text-sm capitalize ${
                kind === k
                  ? 'bg-slate-900 text-white dark:bg-slate-100 dark:text-slate-900'
                  : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800'
              }`}
            >
              {k}s
            </button>
          ))}
        </div>
      </div>
      <TrendBars rows={data.rows} kind={kind} />
    </Card>
  )
}

function NetWorthTab({ range }: { range: DateRange }) {
  const { data } = useQuery({ queryKey: ['report-net-worth', range], queryFn: () => fetchReportNetWorth(range) })
  if (!data) return null
  if (data.rows.length === 0) {
    return (
      <EmptyState>
        No balance snapshots yet. Record account balances on each account's page to build the net-worth history.
      </EmptyState>
    )
  }
  return (
    <Card>
      <p className="mb-2 text-sm text-slate-500 dark:text-slate-400">
        Month-end net worth by account type, carrying each account's latest snapshot forward.
      </p>
      <NetWorthArea rows={data.rows} />
    </Card>
  )
}

function CategoriesTab({ range }: { range: DateRange }) {
  const { data } = useQuery({ queryKey: ['report-summary', range], queryFn: () => fetchReportSummary(range) })
  const { data: trends } = useQuery({ queryKey: ['report-trends', range], queryFn: () => fetchReportTrends(range) })

  // Full expense breakdown from the trends matrix (summary only has top 5).
  const flows = useMemo(() => {
    if (!trends) return []
    const totals = new Map<string, { color: string; cents: number }>()
    for (const r of trends.rows) {
      const isExpense = r.kind === 'expense' || (r.kind === '' && r.outflowCents > 0)
      if (!isExpense) continue
      const name = r.kind === '' ? 'Uncategorized' : r.name
      const cur = totals.get(name) ?? { color: r.color, cents: 0 }
      cur.cents += r.kind === 'expense' ? r.outflowCents - r.inflowCents : r.outflowCents
      totals.set(name, cur)
    }
    return [...totals.entries()]
      .filter(([, v]) => v.cents > 0)
      .sort((a, b) => b[1].cents - a[1].cents)
      .map(([name, v]) => ({ categoryId: null, name, kind: 'expense' as const, color: v.color, amountCents: v.cents }))
  }, [trends])

  if (!data || !trends) return null
  if (flows.length === 0) {
    return <EmptyState>No expenses in this period.</EmptyState>
  }

  const total = flows.reduce((sum, f) => sum + f.amountCents, 0)

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CategoryDonut flows={flows.slice(0, 12)} />
      </Card>
      <Card>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-200 dark:border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
              <th className="py-2 font-medium">Category</th>
              <th className="py-2 text-right font-medium">Spent</th>
              <th className="py-2 text-right font-medium">Share</th>
            </tr>
          </thead>
          <tbody>
            {flows.map((f) => (
              <tr key={f.name} className="border-b border-slate-100 dark:border-slate-800 last:border-b-0">
                <td className="flex items-center gap-2 py-2 text-slate-900 dark:text-slate-100">
                  <span className="inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: f.color || '#898781' }} />
                  {f.name}
                </td>
                <td className="py-2 text-right tabular-nums text-slate-900 dark:text-slate-100">
                  {formatCents(f.amountCents)}
                </td>
                <td className="py-2 text-right tabular-nums text-slate-500 dark:text-slate-400">
                  {total > 0 ? Math.round((f.amountCents / total) * 100) : 0}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  )
}
