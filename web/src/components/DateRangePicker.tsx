import { useState } from 'react'
import type { DateRange } from '../api'

function iso(d: Date): string {
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${day}`
}

export const rangePresets: { key: string; label: string; range: () => DateRange }[] = [
  {
    key: 'this-month',
    label: 'This month',
    range: () => {
      const now = new Date()
      return { from: iso(new Date(now.getFullYear(), now.getMonth(), 1)), to: iso(now) }
    },
  },
  {
    key: 'last-month',
    label: 'Last month',
    range: () => {
      const now = new Date()
      return {
        from: iso(new Date(now.getFullYear(), now.getMonth() - 1, 1)),
        to: iso(new Date(now.getFullYear(), now.getMonth(), 0)),
      }
    },
  },
  {
    key: 'ytd',
    label: 'Year to date',
    range: () => {
      const now = new Date()
      return { from: iso(new Date(now.getFullYear(), 0, 1)), to: iso(now) }
    },
  },
  {
    key: '12m',
    label: 'Last 12 months',
    range: () => {
      const now = new Date()
      const from = new Date(now)
      from.setFullYear(from.getFullYear() - 1)
      return { from: iso(from), to: iso(now) }
    },
  },
]

export function DateRangePicker({
  value,
  preset,
  onChange,
}: {
  value: DateRange
  preset: string | null
  onChange: (range: DateRange, preset: string | null) => void
}) {
  const [custom, setCustom] = useState(false)

  return (
    <div className="flex flex-wrap items-center gap-2">
      {rangePresets.map((p) => (
        <button
          key={p.key}
          type="button"
          onClick={() => {
            setCustom(false)
            onChange(p.range(), p.key)
          }}
          className={`rounded-lg border px-3 py-1.5 text-sm ${
            preset === p.key && !custom
              ? 'border-slate-900 bg-slate-900 text-white dark:border-slate-100 dark:bg-slate-100 dark:text-slate-900'
              : 'border-slate-300 text-slate-600 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-800'
          }`}
        >
          {p.label}
        </button>
      ))}
      <button
        type="button"
        onClick={() => setCustom(true)}
        className={`rounded-lg border px-3 py-1.5 text-sm ${
          custom || preset === null
            ? 'border-slate-900 bg-slate-900 text-white dark:border-slate-100 dark:bg-slate-100 dark:text-slate-900'
            : 'border-slate-300 text-slate-600 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-800'
        }`}
      >
        Custom
      </button>
      {(custom || preset === null) && (
        <span className="flex items-center gap-2">
          <input
            type="date"
            className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-sm"
            value={value.from}
            onChange={(e) => onChange({ ...value, from: e.target.value }, null)}
          />
          <span className="text-sm text-slate-400">to</span>
          <input
            type="date"
            className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-sm"
            value={value.to}
            onChange={(e) => onChange({ ...value, to: e.target.value }, null)}
          />
        </span>
      )}
    </div>
  )
}
