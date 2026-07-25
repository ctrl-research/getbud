import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  fetchContributionRoom,
  fetchReportNetWorth,
  fetchReportSummary,
  fetchTransactions,
  type DateRange,
} from '../api'
import { rangePresets } from '../components/DateRangePicker'
import { MoneyAmount } from '../components/ui'
import { TransactionTable } from '../components/TransactionTable'
import { formatCents } from '../money'
import { NetWorthArea } from '../charts/NetWorthArea'

export function DashboardPage() {
  const thisMonth: DateRange = rangePresets[0].range()
  const yearRange: DateRange = rangePresets[3].range()

  const { data: summary } = useQuery({
    queryKey: ['report-summary', thisMonth],
    queryFn: () => fetchReportSummary(thisMonth),
  })
  const { data: netWorth } = useQuery({
    queryKey: ['report-net-worth', yearRange],
    queryFn: () => fetchReportNetWorth(yearRange),
  })
  const { data: room } = useQuery({
    queryKey: ['contribution-room', new Date().getFullYear()],
    queryFn: () => fetchContributionRoom(),
  })
  const { data: recent } = useQuery({
    queryKey: ['transactions', 'recent'],
    queryFn: () => fetchTransactions({ limit: 8 }),
  })

  const latestNetWorth = netWorth?.rows.length
    ? netWorth.rows
        .filter((r) => r.month === netWorth.rows[netWorth.rows.length - 1].month)
        .reduce((sum, r) => sum + r.totalCents, 0)
    : null

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8">
      <h1 className="mb-6 text-2xl font-semibold text-slate-900 dark:text-slate-100">Dashboard</h1>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatTile label="Net worth" value={latestNetWorth} hint={latestNetWorth === null ? 'add balance snapshots' : ''} />
        <StatTile label="Income this month" value={summary?.incomeCents ?? null} />
        <StatTile label="Spent this month" value={summary?.expenseCents ?? null} />
        <StatTile label="Net this month" value={summary?.netCents ?? null} signed />
      </div>

      {summary && summary.uncategorizedCount > 0 && (
        <Link
          to="/transactions"
          className="mb-6 block rounded-xl border border-amber-200 dark:border-amber-900 bg-amber-50 dark:bg-amber-950/40 px-4 py-3 text-sm text-amber-800 dark:text-amber-200 hover:bg-amber-100 dark:hover:bg-amber-950/70"
        >
          {summary.uncategorizedCount} transaction{summary.uncategorizedCount === 1 ? '' : 's'} this month{' '}
          {summary.uncategorizedCount === 1 ? 'needs' : 'need'} a category — categorize them to keep reports accurate.
        </Link>
      )}

      <div className="mb-6 grid gap-4 lg:grid-cols-3">
        <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4 lg:col-span-2">
          <div className="mb-1 flex items-baseline justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
              Net worth · last 12 months
            </h2>
            <Link to="/reports" className="text-xs text-slate-500 underline hover:text-slate-900 dark:hover:text-slate-100">
              Reports →
            </Link>
          </div>
          {netWorth && netWorth.rows.length > 0 ? (
            <NetWorthArea rows={netWorth.rows} />
          ) : (
            <p className="py-16 text-center text-sm text-slate-400 dark:text-slate-500">
              Record balance snapshots on your accounts to build this chart.
            </p>
          )}
        </div>

        <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4">
          <div className="mb-3 flex items-baseline justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
              {new Date().getFullYear()} contribution room
            </h2>
            <Link
              to="/contributions"
              className="text-xs text-slate-500 underline hover:text-slate-900 dark:hover:text-slate-100"
            >
              Details →
            </Link>
          </div>
          <div className="space-y-4">
            {(['tfsa', 'rrsp', 'fhsa'] as const).map((type) => {
              const info = room?.types[type]
              if (!info) return null
              const pct =
                info.roomCents && info.roomCents > 0
                  ? Math.min(100, Math.round((info.contributedCents / info.roomCents) * 100))
                  : null
              return (
                <div key={type}>
                  <div className="mb-1 flex justify-between text-sm">
                    <span className="uppercase text-slate-700 dark:text-slate-300">{type}</span>
                    <span className="tabular-nums text-slate-500 dark:text-slate-400">
                      {formatCents(info.contributedCents)}
                      {info.roomCents !== null && ` / ${formatCents(info.roomCents)}`}
                    </span>
                  </div>
                  {pct !== null ? (
                    <div className="h-2 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
                      <div className="h-full rounded-full bg-teal-600" style={{ width: `${pct}%` }} />
                    </div>
                  ) : (
                    <p className="text-xs text-slate-400 dark:text-slate-500">room not set</p>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      </div>

      <section>
        <div className="mb-2 flex items-baseline justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
            Recent transactions
          </h2>
          <Link to="/transactions" className="text-xs text-slate-500 underline hover:text-slate-900 dark:hover:text-slate-100">
            All transactions →
          </Link>
        </div>
        {recent && recent.transactions.length > 0 ? (
          <TransactionTable transactions={recent.transactions} />
        ) : (
          <p className="rounded-xl border border-dashed border-slate-300 dark:border-slate-700 p-10 text-center text-sm text-slate-500 dark:text-slate-400">
            Nothing yet — add accounts, then add or import transactions to get started.
          </p>
        )}
      </section>
    </div>
  )
}

function StatTile({
  label,
  value,
  hint = '',
  signed = false,
}: {
  label: string
  value: number | null
  hint?: string
  signed?: boolean
}) {
  const color = !signed
    ? 'text-slate-900 dark:text-slate-100'
    : (value ?? 0) < 0
      ? 'text-red-600 dark:text-red-400'
      : 'text-emerald-600 dark:text-emerald-400'
  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4">
      <div className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">{label}</div>
      <div className={`mt-1 text-2xl font-semibold ${color}`}>
        {value !== null ? <MoneyAmount cents={value} /> : <span className="text-sm font-normal text-slate-400">{hint || '—'}</span>}
      </div>
    </div>
  )
}
