import type { ReactNode } from 'react'
import { formatCents } from '../money'

/** Simple centered modal; render conditionally. */
export function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/40 p-4 pt-16"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="w-full max-w-md rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-6 shadow-lg">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">{title}</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="rounded p-1 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
          >
            ✕
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}

export function MoneyAmount({ cents, currency = 'CAD', colored = false }: { cents: number; currency?: string; colored?: boolean }) {
  const cls = !colored
    ? 'text-slate-900 dark:text-slate-100'
    : cents < 0
      ? 'text-red-600 dark:text-red-400'
      : 'text-emerald-600 dark:text-emerald-400'
  return <span className={`tabular-nums ${cls}`}>{formatCents(cents, currency)}</span>
}

export const inputCls =
  'mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 focus:border-slate-500 focus:outline-none'

export const labelCls = 'text-sm font-medium text-slate-700 dark:text-slate-300'

export const primaryBtn =
  'rounded-lg bg-slate-900 dark:bg-slate-100 px-4 py-2 text-sm font-medium text-white dark:text-slate-900 hover:bg-slate-700 dark:hover:bg-slate-300 disabled:opacity-50'

export const secondaryBtn =
  'rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-2 text-sm text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-50'

export const dangerBtn =
  'rounded-lg border border-red-300 dark:border-red-800 px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950 disabled:opacity-50'

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className={labelCls}>{label}</span>
      {children}
    </label>
  )
}

export function CategoryDot({ color }: { color: string }) {
  return (
    <span
      className="inline-block h-2.5 w-2.5 shrink-0 rounded-full"
      style={{ backgroundColor: color || '#94a3b8' }}
    />
  )
}

export function EmptyState({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-xl border border-dashed border-slate-300 dark:border-slate-700 p-10 text-center text-sm text-slate-500 dark:text-slate-400">
      {children}
    </div>
  )
}
