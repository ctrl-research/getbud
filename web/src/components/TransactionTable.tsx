import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  deleteTransaction,
  fetchCategories,
  updateTransaction,
  type Transaction,
} from '../api'
import { CategoryDot, MoneyAmount } from './ui'

/** Shared transaction list with inline category editing and delete. */
export function TransactionTable({
  transactions,
  hideAccount = false,
}: {
  transactions: Transaction[]
  hideAccount?: boolean
}) {
  const queryClient = useQueryClient()
  const remove = useMutation({
    mutationFn: deleteTransaction,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
    },
  })

  return (
    <div className="overflow-x-auto rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 dark:border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
            <th className="px-4 py-2 font-medium">Date</th>
            <th className="px-4 py-2 font-medium">Payee</th>
            {!hideAccount && <th className="px-4 py-2 font-medium">Account</th>}
            <th className="px-4 py-2 font-medium">Category</th>
            <th className="px-4 py-2 text-right font-medium">Amount</th>
            <th className="px-2 py-2" />
          </tr>
        </thead>
        <tbody>
          {transactions.map((t) => (
            <tr key={t.id} className="border-b border-slate-100 dark:border-slate-800 last:border-b-0">
              <td className="whitespace-nowrap px-4 py-2 text-slate-600 dark:text-slate-300">{t.date}</td>
              <td className="max-w-56 truncate px-4 py-2 text-slate-900 dark:text-slate-100" title={t.notes || t.payee}>
                {t.payee || (t.isTransfer ? 'Transfer' : '—')}
                {t.notes && <span className="ml-2 text-xs text-slate-400">{t.notes}</span>}
              </td>
              {!hideAccount && (
                <td className="whitespace-nowrap px-4 py-2 text-slate-500 dark:text-slate-400">{t.accountName}</td>
              )}
              <td className="px-4 py-2">
                <CategoryCell transaction={t} />
              </td>
              <td className="whitespace-nowrap px-4 py-2 text-right">
                <MoneyAmount cents={t.amountCents} colored={!t.isTransfer} />
              </td>
              <td className="px-2 py-2 text-right">
                <button
                  type="button"
                  aria-label="Delete transaction"
                  className="px-1 text-slate-300 hover:text-red-600 dark:text-slate-600 dark:hover:text-red-400"
                  onClick={() => {
                    if (window.confirm(t.isTransfer ? 'Delete both legs of this transfer?' : 'Delete this transaction?'))
                      remove.mutate(t.id)
                  }}
                >
                  ✕
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function CategoryCell({ transaction: t }: { transaction: Transaction }) {
  const queryClient = useQueryClient()
  const { data: categories } = useQuery({ queryKey: ['categories'], queryFn: fetchCategories })
  const [editing, setEditing] = useState(false)

  const update = useMutation({
    mutationFn: (categoryId: string | null) =>
      updateTransaction(t.id, categoryId === null ? { clearCategory: true } : { categoryId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      setEditing(false)
    },
  })

  if (t.isTransfer) {
    return <span className="text-xs text-slate-400 dark:text-slate-500">Transfer</span>
  }

  if (editing && categories) {
    const active = categories.filter((c) => !c.isArchived)
    return (
      <select
        autoFocus
        className="rounded border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-1 py-0.5 text-xs"
        value={t.categoryId ?? ''}
        onChange={(e) => update.mutate(e.target.value === '' ? null : e.target.value)}
        onBlur={() => setEditing(false)}
      >
        <option value="">Uncategorized</option>
        <optgroup label="Expense">
          {active.filter((c) => c.kind === 'expense').map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </optgroup>
        <optgroup label="Income">
          {active.filter((c) => c.kind === 'income').map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </optgroup>
      </select>
    )
  }

  return (
    <button
      type="button"
      className="flex items-center gap-1.5 rounded px-1 py-0.5 text-xs hover:bg-slate-100 dark:hover:bg-slate-800"
      onClick={() => setEditing(true)}
      title="Change category"
    >
      {t.categoryName ? (
        <>
          <CategoryDot color={t.categoryColor ?? ''} />
          <span className="text-slate-700 dark:text-slate-300">{t.categoryName}</span>
        </>
      ) : (
        <span className="italic text-amber-600 dark:text-amber-400">Uncategorized</span>
      )}
    </button>
  )
}
