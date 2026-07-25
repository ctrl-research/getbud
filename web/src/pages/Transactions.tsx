import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createTransaction,
  createTransfer,
  fetchAccounts,
  fetchCategories,
  fetchTransactions,
  type CategoryKind,
  type TransactionFilters,
} from '../api'
import { parseCents, todayISO } from '../money'
import { EmptyState, Field, Modal, inputCls, primaryBtn, secondaryBtn } from '../components/ui'
import { TransactionTable } from '../components/TransactionTable'

const PAGE_SIZE = 50

export function TransactionsPage() {
  const [filters, setFilters] = useState<TransactionFilters>({})
  const [page, setPage] = useState(0)
  const [adding, setAdding] = useState<'transaction' | 'transfer' | null>(null)

  const { data: accounts } = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const { data: categories } = useQuery({ queryKey: ['categories'], queryFn: fetchCategories })
  const { data } = useQuery({
    queryKey: ['transactions', filters, page],
    queryFn: () => fetchTransactions({ ...filters, limit: PAGE_SIZE, offset: page * PAGE_SIZE }),
  })

  const setFilter = (patch: Partial<TransactionFilters>) => {
    setFilters((f) => ({ ...f, ...patch }))
    setPage(0)
  }

  const totalPages = data ? Math.max(1, Math.ceil(data.totalCount / PAGE_SIZE)) : 1

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">Transactions</h1>
        <div className="flex gap-2">
          <button type="button" className={secondaryBtn} onClick={() => setAdding('transfer')}>
            Add transfer
          </button>
          <button type="button" className={primaryBtn} onClick={() => setAdding('transaction')}>
            Add transaction
          </button>
        </div>
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <input
          className="w-56 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-3 py-1.5 text-sm"
          placeholder="Search payee or notes…"
          value={filters.q ?? ''}
          onChange={(e) => setFilter({ q: e.target.value || undefined })}
        />
        <input
          type="date"
          className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-sm"
          value={filters.from ?? ''}
          onChange={(e) => setFilter({ from: e.target.value || undefined })}
        />
        <span className="text-sm text-slate-400">to</span>
        <input
          type="date"
          className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-sm"
          value={filters.to ?? ''}
          onChange={(e) => setFilter({ to: e.target.value || undefined })}
        />
        <select
          className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-sm"
          value={filters.accountId ?? ''}
          onChange={(e) => setFilter({ accountId: e.target.value || undefined })}
        >
          <option value="">All accounts</option>
          {accounts?.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>
        <select
          className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-sm"
          value={filters.uncategorized ? 'uncategorized' : (filters.categoryId ?? '')}
          onChange={(e) => {
            const v = e.target.value
            if (v === 'uncategorized') setFilter({ uncategorized: true, categoryId: undefined })
            else setFilter({ uncategorized: undefined, categoryId: v || undefined })
          }}
        >
          <option value="">All categories</option>
          <option value="uncategorized">Uncategorized</option>
          {categories
            ?.filter((c) => !c.isArchived)
            .map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
        </select>
        {(filters.q || filters.from || filters.to || filters.accountId || filters.categoryId || filters.uncategorized) && (
          <button
            type="button"
            className="text-sm text-slate-500 underline hover:text-slate-900 dark:hover:text-slate-100"
            onClick={() => setFilter({ q: undefined, from: undefined, to: undefined, accountId: undefined, categoryId: undefined, uncategorized: undefined })}
          >
            Clear
          </button>
        )}
      </div>

      {data && data.transactions.length === 0 ? (
        <EmptyState>
          No transactions found. Add one manually or import a CSV from your bank on the Import page.
        </EmptyState>
      ) : (
        data && <TransactionTable transactions={data.transactions} />
      )}

      {data && data.totalCount > PAGE_SIZE && (
        <div className="mt-4 flex items-center justify-between text-sm text-slate-500 dark:text-slate-400">
          <span>
            {data.totalCount} transactions · page {page + 1} of {totalPages}
          </span>
          <div className="flex gap-2">
            <button type="button" className={secondaryBtn} disabled={page === 0} onClick={() => setPage((p) => p - 1)}>
              Previous
            </button>
            <button
              type="button"
              className={secondaryBtn}
              disabled={page + 1 >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
            </button>
          </div>
        </div>
      )}

      {adding === 'transaction' && <AddTransactionModal onClose={() => setAdding(null)} />}
      {adding === 'transfer' && <AddTransferModal onClose={() => setAdding(null)} />}
    </div>
  )
}

function AddTransactionModal({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient()
  const { data: accounts } = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const { data: categories } = useQuery({ queryKey: ['categories'], queryFn: fetchCategories })

  const [kind, setKind] = useState<CategoryKind>('expense')
  const [accountId, setAccountId] = useState('')
  const [date, setDate] = useState(todayISO())
  const [amount, setAmount] = useState('')
  const [payee, setPayee] = useState('')
  const [notes, setNotes] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [error, setError] = useState('')

  const activeAccounts = accounts?.filter((a) => !a.isArchived) ?? []
  const kindCategories = categories?.filter((c) => !c.isArchived && c.kind === kind) ?? []

  const mutation = useMutation({
    mutationFn: () => {
      const cents = parseCents(amount)
      if (cents === null || cents <= 0) throw new Error('Enter a positive amount')
      return createTransaction({
        accountId,
        date,
        // Sign comes from the income/expense toggle; users always type a
        // positive number.
        amountCents: kind === 'expense' ? -cents : cents,
        payee: payee.trim(),
        notes: notes.trim(),
        categoryId: categoryId || null,
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['transactions'] })
      await queryClient.invalidateQueries({ queryKey: ['accounts'] })
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to add transaction'),
  })

  return (
    <Modal title="Add transaction" onClose={onClose}>
      <form
        className="space-y-3"
        onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate()
        }}
      >
        <div className="flex gap-2">
          {(['expense', 'income'] as const).map((k) => (
            <button
              key={k}
              type="button"
              onClick={() => {
                setKind(k)
                setCategoryId('')
              }}
              className={`flex-1 rounded-lg border px-3 py-1.5 text-sm capitalize ${
                kind === k
                  ? 'border-slate-900 bg-slate-900 text-white dark:border-slate-100 dark:bg-slate-100 dark:text-slate-900'
                  : 'border-slate-300 text-slate-600 dark:border-slate-600 dark:text-slate-400'
              }`}
            >
              {k}
            </button>
          ))}
        </div>
        <Field label="Account">
          <select className={inputCls} required value={accountId} onChange={(e) => setAccountId(e.target.value)}>
            <option value="" disabled>
              Select account…
            </option>
            {activeAccounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Date">
            <input type="date" className={inputCls} required value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          <Field label="Amount">
            <input
              className={inputCls}
              required
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="12.34"
              inputMode="decimal"
            />
          </Field>
        </div>
        <Field label="Payee">
          <input className={inputCls} value={payee} onChange={(e) => setPayee(e.target.value)} placeholder="Loblaws" />
        </Field>
        <Field label="Category">
          <select className={inputCls} value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
            <option value="">Uncategorized</option>
            {kindCategories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Notes (optional)">
          <input className={inputCls} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </Field>
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        <button type="submit" disabled={mutation.isPending || !accountId} className={`w-full ${primaryBtn}`}>
          {mutation.isPending ? 'Adding…' : 'Add transaction'}
        </button>
      </form>
    </Modal>
  )
}

function AddTransferModal({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient()
  const { data: accounts } = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const [fromId, setFromId] = useState('')
  const [toId, setToId] = useState('')
  const [date, setDate] = useState(todayISO())
  const [amount, setAmount] = useState('')
  const [notes, setNotes] = useState('')
  const [error, setError] = useState('')

  const activeAccounts = accounts?.filter((a) => !a.isArchived) ?? []

  const mutation = useMutation({
    mutationFn: () => {
      const cents = parseCents(amount)
      if (cents === null || cents <= 0) throw new Error('Enter a positive amount')
      if (fromId === toId) throw new Error('Choose two different accounts')
      return createTransfer({ fromAccountId: fromId, toAccountId: toId, date, amountCents: cents, notes: notes.trim() })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['transactions'] })
      await queryClient.invalidateQueries({ queryKey: ['accounts'] })
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to add transfer'),
  })

  return (
    <Modal title="Add transfer" onClose={onClose}>
      <p className="mb-3 text-xs text-slate-500 dark:text-slate-400">
        Transfers move money between your own accounts and never count as income or expenses. A transfer into an RRSP,
        TFSA, or FHSA counts toward that year's contributions.
      </p>
      <form
        className="space-y-3"
        onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate()
        }}
      >
        <Field label="From">
          <select className={inputCls} required value={fromId} onChange={(e) => setFromId(e.target.value)}>
            <option value="" disabled>
              Select account…
            </option>
            {activeAccounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label="To">
          <select className={inputCls} required value={toId} onChange={(e) => setToId(e.target.value)}>
            <option value="" disabled>
              Select account…
            </option>
            {activeAccounts
              .filter((a) => a.id !== fromId)
              .map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
          </select>
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Date">
            <input type="date" className={inputCls} required value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          <Field label="Amount">
            <input
              className={inputCls}
              required
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="500.00"
              inputMode="decimal"
            />
          </Field>
        </div>
        <Field label="Notes (optional)">
          <input className={inputCls} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </Field>
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        <button type="submit" disabled={mutation.isPending || !fromId || !toId} className={`w-full ${primaryBtn}`}>
          {mutation.isPending ? 'Adding…' : 'Add transfer'}
        </button>
      </form>
    </Modal>
  )
}
