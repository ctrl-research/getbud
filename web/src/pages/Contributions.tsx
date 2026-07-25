import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchContributionRoom, putContributionRoom, type ContributionType } from '../api'
import { centsToInput, formatCents, parseCents } from '../money'
import { inputCls, primaryBtn, secondaryBtn } from '../components/ui'

const TYPE_LABELS: Record<'rrsp' | 'tfsa' | 'fhsa', { name: string; blurb: string }> = {
  tfsa: { name: 'TFSA', blurb: 'Room is cumulative and personal — check CRA MyAccount for your number.' },
  rrsp: { name: 'RRSP', blurb: 'Deduction limit is on your latest notice of assessment.' },
  fhsa: { name: 'FHSA', blurb: '$8,000/year participation room, $40,000 lifetime.' },
}

export function ContributionsPage() {
  const currentYear = new Date().getFullYear()
  const [year, setYear] = useState(currentYear)
  const { data } = useQuery({
    queryKey: ['contribution-room', year],
    queryFn: () => fetchContributionRoom(year),
  })

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8">
      <div className="mb-2 flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">Contributions</h1>
        <div className="flex items-center gap-2">
          <button type="button" className={secondaryBtn} onClick={() => setYear((y) => y - 1)}>
            ←
          </button>
          <span className="w-16 text-center text-lg font-medium text-slate-900 dark:text-slate-100">{year}</span>
          <button
            type="button"
            className={secondaryBtn}
            disabled={year >= currentYear}
            onClick={() => setYear((y) => y + 1)}
          >
            →
          </button>
        </div>
      </div>
      <p className="mb-6 text-sm text-slate-500 dark:text-slate-400">
        Contributions and withdrawals are derived from transactions in your RRSP/TFSA/FHSA accounts for the calendar
        year. Enter your available room from CRA — it's personal and can't be computed automatically. (Note: the RRSP
        first-60-days rule isn't modelled; contributions count toward the calendar year they occur in.)
      </p>

      <div className="grid gap-4 md:grid-cols-3">
        {(['tfsa', 'rrsp', 'fhsa'] as const).map((type) =>
          data ? <TypeCard key={type} type={type} year={year} info={data.types[type]} /> : null,
        )}
      </div>
    </div>
  )
}

function TypeCard({ type, year, info }: { type: 'rrsp' | 'tfsa' | 'fhsa'; year: number; info: ContributionType }) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [room, setRoom] = useState('')
  const [error, setError] = useState('')

  const save = useMutation({
    mutationFn: () => {
      const cents = parseCents(room)
      if (cents === null || cents < 0) throw new Error('Enter a valid amount')
      return putContributionRoom(type, year, cents, info.notes)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['contribution-room'] })
      setEditing(false)
      setError('')
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to save'),
  })

  const label = TYPE_LABELS[type]
  const pct =
    info.roomCents && info.roomCents > 0
      ? Math.min(100, Math.round((info.contributedCents / info.roomCents) * 100))
      : null
  const over = info.remainingCents !== null && info.remainingCents < 0

  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-5">
      <div className="flex items-baseline justify-between">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">{label.name}</h2>
        <span className="text-xs text-slate-400">{year}</span>
      </div>

      <dl className="mt-4 space-y-2 text-sm">
        <div className="flex justify-between">
          <dt className="text-slate-500 dark:text-slate-400">Contributed</dt>
          <dd className="tabular-nums text-slate-900 dark:text-slate-100">{formatCents(info.contributedCents)}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-slate-500 dark:text-slate-400">Withdrawn</dt>
          <dd className="tabular-nums text-slate-900 dark:text-slate-100">{formatCents(info.withdrawnCents)}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-slate-500 dark:text-slate-400">Room</dt>
          <dd className="tabular-nums text-slate-900 dark:text-slate-100">
            {info.roomCents !== null ? formatCents(info.roomCents) : <span className="text-slate-400">not set</span>}
          </dd>
        </div>
        {info.remainingCents !== null && (
          <div className="flex justify-between font-medium">
            <dt className={over ? 'text-red-600 dark:text-red-400' : 'text-slate-700 dark:text-slate-300'}>
              {over ? 'Over-contributed' : 'Remaining'}
            </dt>
            <dd className={`tabular-nums ${over ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'}`}>
              {formatCents(Math.abs(info.remainingCents))}
            </dd>
          </div>
        )}
      </dl>

      {pct !== null && (
        <div className="mt-3 h-2 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
          <div
            className={`h-full rounded-full ${over ? 'bg-red-500' : 'bg-teal-600'}`}
            style={{ width: `${pct}%` }}
          />
        </div>
      )}

      <div className="mt-4">
        {editing ? (
          <form
            className="space-y-2"
            onSubmit={(e) => {
              e.preventDefault()
              save.mutate()
            }}
          >
            <input
              className={inputCls}
              autoFocus
              value={room}
              onChange={(e) => setRoom(e.target.value)}
              placeholder={info.defaultHintCents ? `e.g. ${centsToInput(info.defaultHintCents)} (this year's limit)` : '0.00'}
              inputMode="decimal"
            />
            {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
            <div className="flex gap-2">
              <button type="submit" className={`flex-1 ${primaryBtn}`} disabled={save.isPending}>
                Save
              </button>
              <button type="button" className={secondaryBtn} onClick={() => setEditing(false)}>
                Cancel
              </button>
            </div>
          </form>
        ) : (
          <button
            type="button"
            className={`w-full ${secondaryBtn}`}
            onClick={() => {
              setRoom(info.roomCents !== null ? centsToInput(info.roomCents) : '')
              setEditing(true)
            }}
          >
            {info.roomCents !== null ? 'Update room' : 'Set room'}
          </button>
        )}
        <p className="mt-2 text-xs text-slate-400 dark:text-slate-500">{label.blurb}</p>
      </div>
    </div>
  )
}
