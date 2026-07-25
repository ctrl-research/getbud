import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  accountTypeLabels,
  createAccount,
  fetchAccounts,
  type Account,
  type AccountType,
} from '../api'
import { parseCents } from '../money'
import { EmptyState, Field, Modal, MoneyAmount, inputCls, primaryBtn } from '../components/ui'

export function AccountsPage() {
  const { data: accounts } = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const [adding, setAdding] = useState(false)

  if (!accounts) return null

  const active = accounts.filter((a) => !a.isArchived)
  const archived = accounts.filter((a) => a.isArchived)
  const cash = active.filter((a) => !a.isInvestment)
  const investment = active.filter((a) => a.isInvestment)

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">Accounts</h1>
        <button type="button" className={primaryBtn} onClick={() => setAdding(true)}>
          Add account
        </button>
      </div>

      {active.length === 0 && (
        <EmptyState>No accounts yet. Add your chequing, credit card, and investment accounts to get started.</EmptyState>
      )}

      {cash.length > 0 && <AccountGroup title="Cash & credit" accounts={cash} />}
      {investment.length > 0 && <AccountGroup title="Investments" accounts={investment} />}
      {archived.length > 0 && <AccountGroup title="Archived" accounts={archived} muted />}

      {adding && <AddAccountModal onClose={() => setAdding(false)} />}
    </div>
  )
}

function AccountGroup({ title, accounts, muted = false }: { title: string; accounts: Account[]; muted?: boolean }) {
  const total = accounts.reduce((sum, a) => sum + a.balanceCents, 0)
  return (
    <section className={`mb-8 ${muted ? 'opacity-60' : ''}`}>
      <div className="mb-2 flex items-baseline justify-between">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">{title}</h2>
        <MoneyAmount cents={total} />
      </div>
      <div className="overflow-hidden rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900">
        {accounts.map((a, i) => (
          <Link
            key={a.id}
            to="/accounts/$accountId"
            params={{ accountId: a.id }}
            className={`flex items-center justify-between px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800 ${
              i > 0 ? 'border-t border-slate-100 dark:border-slate-800' : ''
            }`}
          >
            <div>
              <div className="text-sm font-medium text-slate-900 dark:text-slate-100">{a.name}</div>
              <div className="text-xs text-slate-500 dark:text-slate-400">
                {accountTypeLabels[a.type]}
                {a.institution ? ` · ${a.institution}` : ''}
                {a.currency !== 'CAD' ? ` · ${a.currency}` : ''}
              </div>
            </div>
            <MoneyAmount cents={a.balanceCents} currency={a.currency} />
          </Link>
        ))}
      </div>
    </section>
  )
}

function AddAccountModal({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [type, setType] = useState<AccountType>('chequing')
  const [institution, setInstitution] = useState('')
  const [opening, setOpening] = useState('')
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => {
      const openingCents = opening.trim() === '' ? 0 : parseCents(opening)
      if (openingCents === null) throw new Error('Opening balance must be a valid amount')
      return createAccount({
        name: name.trim(),
        type,
        institution: institution.trim(),
        openingBalanceCents: openingCents,
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['accounts'] })
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to create account'),
  })

  const isInvestment = ['rrsp', 'tfsa', 'fhsa', 'non_registered'].includes(type)

  return (
    <Modal title="Add account" onClose={onClose}>
      <form
        className="space-y-3"
        onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate()
        }}
      >
        <Field label="Name">
          <input className={inputCls} required value={name} onChange={(e) => setName(e.target.value)} placeholder="RBC Chequing" />
        </Field>
        <Field label="Type">
          <select className={inputCls} value={type} onChange={(e) => setType(e.target.value as AccountType)}>
            {Object.entries(accountTypeLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Institution (optional)">
          <input className={inputCls} value={institution} onChange={(e) => setInstitution(e.target.value)} placeholder="RBC" />
        </Field>
        {!isInvestment && (
          <Field label="Opening balance (optional)">
            <input className={inputCls} value={opening} onChange={(e) => setOpening(e.target.value)} placeholder="0.00" inputMode="decimal" />
          </Field>
        )}
        {isInvestment && (
          <p className="text-xs text-slate-500 dark:text-slate-400">
            Investment balances are tracked with balance snapshots — add one from the account page after creating it.
          </p>
        )}
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        <button type="submit" disabled={mutation.isPending} className={`w-full ${primaryBtn}`}>
          {mutation.isPending ? 'Creating…' : 'Create account'}
        </button>
      </form>
    </Modal>
  )
}
