import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import {
  accountTypeLabels,
  deleteAccount,
  deleteSnapshot,
  fetchAccounts,
  fetchSnapshots,
  fetchTransactions,
  updateAccount,
  upsertSnapshot,
} from '../api'
import { centsToInput, parseCents, todayISO } from '../money'
import { EmptyState, Field, Modal, MoneyAmount, dangerBtn, inputCls, primaryBtn, secondaryBtn } from '../components/ui'
import { TransactionTable } from '../components/TransactionTable'

export function AccountDetailPage() {
  const { accountId } = useParams({ from: '/accounts/$accountId' })
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: accounts } = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const { data: txns } = useQuery({
    queryKey: ['transactions', { accountId }],
    queryFn: () => fetchTransactions({ accountId, limit: 50 }),
  })
  const [editing, setEditing] = useState(false)

  const archive = useMutation({
    mutationFn: (isArchived: boolean) => updateAccount(accountId, { isArchived }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['accounts'] }),
  })
  const remove = useMutation({
    mutationFn: () => deleteAccount(accountId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['accounts'] })
      navigate({ to: '/accounts' })
    },
  })

  const account = accounts?.find((a) => a.id === accountId)
  if (!account) return null

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">
            {account.name}
            {account.isArchived && (
              <span className="ml-2 rounded bg-slate-200 dark:bg-slate-700 px-2 py-0.5 text-xs font-normal text-slate-600 dark:text-slate-300">
                archived
              </span>
            )}
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {accountTypeLabels[account.type]}
            {account.institution ? ` · ${account.institution}` : ''} · {account.currency}
          </p>
        </div>
        <div className="text-right">
          <div className="text-2xl font-semibold">
            <MoneyAmount cents={account.balanceCents} currency={account.currency} />
          </div>
          <div className="mt-2 flex justify-end gap-2">
            <button type="button" className={secondaryBtn} onClick={() => setEditing(true)}>
              Edit
            </button>
            <button
              type="button"
              className={secondaryBtn}
              disabled={archive.isPending}
              onClick={() => archive.mutate(!account.isArchived)}
            >
              {account.isArchived ? 'Unarchive' : 'Archive'}
            </button>
            {(txns?.totalCount ?? 1) === 0 && (
              <button type="button" className={dangerBtn} disabled={remove.isPending} onClick={() => remove.mutate()}>
                Delete
              </button>
            )}
          </div>
        </div>
      </div>

      {account.isInvestment && <SnapshotsSection accountId={accountId} currency={account.currency} />}

      <section>
        <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          Transactions
        </h2>
        {txns && txns.transactions.length === 0 ? (
          <EmptyState>No transactions in this account yet.</EmptyState>
        ) : (
          txns && <TransactionTable transactions={txns.transactions} hideAccount />
        )}
      </section>

      {editing && <EditAccountModal account={account} onClose={() => setEditing(false)} />}
    </div>
  )
}

function SnapshotsSection({ accountId, currency }: { accountId: string; currency: string }) {
  const queryClient = useQueryClient()
  const { data: snapshots } = useQuery({
    queryKey: ['snapshots', accountId],
    queryFn: () => fetchSnapshots(accountId),
  })
  const [asOf, setAsOf] = useState(todayISO())
  const [balance, setBalance] = useState('')
  const [error, setError] = useState('')

  const add = useMutation({
    mutationFn: () => {
      const cents = parseCents(balance)
      if (cents === null) throw new Error('Balance must be a valid amount')
      return upsertSnapshot(accountId, asOf, cents)
    },
    onSuccess: async () => {
      setBalance('')
      setError('')
      await queryClient.invalidateQueries({ queryKey: ['snapshots', accountId] })
      await queryClient.invalidateQueries({ queryKey: ['accounts'] })
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to save snapshot'),
  })
  const remove = useMutation({
    mutationFn: (snapshotId: string) => deleteSnapshot(accountId, snapshotId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['snapshots', accountId] })
      await queryClient.invalidateQueries({ queryKey: ['accounts'] })
    },
  })

  return (
    <section className="mb-8">
      <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Balance snapshots
      </h2>
      <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4">
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            add.mutate()
          }}
        >
          <Field label="As of">
            <input type="date" className={inputCls} value={asOf} onChange={(e) => setAsOf(e.target.value)} required />
          </Field>
          <Field label="Balance">
            <input
              className={inputCls}
              value={balance}
              onChange={(e) => setBalance(e.target.value)}
              placeholder="12345.67"
              inputMode="decimal"
              required
            />
          </Field>
          <button type="submit" className={primaryBtn} disabled={add.isPending}>
            Save snapshot
          </button>
          {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        </form>

        {snapshots && snapshots.length > 0 && (
          <table className="mt-4 w-full text-sm">
            <tbody>
              {snapshots.map((s) => (
                <tr key={s.id} className="border-t border-slate-100 dark:border-slate-800">
                  <td className="py-2 text-slate-600 dark:text-slate-300">{s.asOf}</td>
                  <td className="py-2 text-right">
                    <MoneyAmount cents={s.balanceCents} currency={currency} />
                  </td>
                  <td className="py-2 pl-4 text-right">
                    <button
                      type="button"
                      className="text-xs text-slate-400 hover:text-red-600 dark:hover:text-red-400"
                      onClick={() => remove.mutate(s.id)}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {snapshots && snapshots.length === 0 && (
          <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">
            No snapshots yet — record the account balance from your statement to start the net-worth history.
          </p>
        )}
      </div>
    </section>
  )
}

function EditAccountModal({
  account,
  onClose,
}: {
  account: { id: string; name: string; institution: string; openingBalanceCents: number; isInvestment: boolean }
  onClose: () => void
}) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(account.name)
  const [institution, setInstitution] = useState(account.institution)
  const [opening, setOpening] = useState(centsToInput(account.openingBalanceCents))
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => {
      const openingCents = parseCents(opening)
      if (openingCents === null) throw new Error('Opening balance must be a valid amount')
      return updateAccount(account.id, {
        name: name.trim(),
        institution: institution.trim(),
        openingBalanceCents: openingCents,
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['accounts'] })
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to update account'),
  })

  return (
    <Modal title="Edit account" onClose={onClose}>
      <form
        className="space-y-3"
        onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate()
        }}
      >
        <Field label="Name">
          <input className={inputCls} required value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label="Institution">
          <input className={inputCls} value={institution} onChange={(e) => setInstitution(e.target.value)} />
        </Field>
        {!account.isInvestment && (
          <Field label="Opening balance">
            <input className={inputCls} value={opening} onChange={(e) => setOpening(e.target.value)} inputMode="decimal" />
          </Field>
        )}
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        <button type="submit" disabled={mutation.isPending} className={`w-full ${primaryBtn}`}>
          Save
        </button>
      </form>
    </Modal>
  )
}
