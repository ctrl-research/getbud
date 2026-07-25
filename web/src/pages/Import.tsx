import { useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  commitImport,
  fetchAccounts,
  fetchImports,
  previewImport,
  revertImport,
  type ImportBatch,
  type ImportMapping,
  type ImportPreview,
} from '../api'
import { Field, MoneyAmount, dangerBtn, inputCls, primaryBtn, secondaryBtn } from '../components/ui'

export function ImportPage() {
  const [file, setFile] = useState<File | null>(null)
  const [accountId, setAccountId] = useState('')
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [excluded, setExcluded] = useState<Set<number>>(new Set())
  const [done, setDone] = useState<ImportBatch | null>(null)

  const reset = () => {
    setFile(null)
    setPreview(null)
    setExcluded(new Set())
    setDone(null)
  }

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8">
      <h1 className="mb-6 text-2xl font-semibold text-slate-900 dark:text-slate-100">Import transactions</h1>

      {done ? (
        <SuccessCard batch={done} onReset={reset} />
      ) : preview && file ? (
        <ReviewStep
          file={file}
          accountId={accountId}
          preview={preview}
          excluded={excluded}
          setExcluded={setExcluded}
          onMappingChange={setPreview}
          onBack={reset}
          onDone={setDone}
        />
      ) : (
        <UploadStep
          file={file}
          setFile={setFile}
          accountId={accountId}
          setAccountId={setAccountId}
          onPreview={(p) => {
            setPreview(p)
            // Duplicates start unchecked; the user can re-check legitimate repeats.
            setExcluded(new Set(p.rows.filter((r) => r.status !== 'ok').map((r) => r.index)))
          }}
        />
      )}

      <ImportHistory />
    </div>
  )
}

function UploadStep({
  file,
  setFile,
  accountId,
  setAccountId,
  onPreview,
}: {
  file: File | null
  setFile: (f: File | null) => void
  accountId: string
  setAccountId: (id: string) => void
  onPreview: (p: ImportPreview) => void
}) {
  const { data: accounts } = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const fileInput = useRef<HTMLInputElement>(null)
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => {
      if (!file || !accountId) throw new Error('Choose an account and a CSV file')
      return previewImport(file, accountId)
    },
    onSuccess: onPreview,
    onError: (e) => setError(e instanceof Error ? e.message : 'Preview failed'),
  })

  const activeAccounts = accounts?.filter((a) => !a.isArchived) ?? []

  return (
    <div className="mb-10 max-w-lg rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-6">
      <p className="mb-4 text-sm text-slate-500 dark:text-slate-400">
        Upload a CSV export from your bank or credit card. You'll confirm the column mapping and review duplicates
        before anything is saved.
      </p>
      <div className="space-y-3">
        <Field label="Into account">
          <select className={inputCls} value={accountId} onChange={(e) => setAccountId(e.target.value)}>
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
        <Field label="CSV file">
          <input
            ref={fileInput}
            type="file"
            accept=".csv,text/csv"
            className="mt-1 block w-full text-sm text-slate-600 dark:text-slate-300 file:mr-3 file:rounded-lg file:border-0 file:bg-slate-100 dark:file:bg-slate-800 file:px-3 file:py-1.5 file:text-sm file:text-slate-700 dark:file:text-slate-300"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </Field>
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        <button
          type="button"
          className={primaryBtn}
          disabled={!file || !accountId || mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          {mutation.isPending ? 'Analyzing…' : 'Preview import'}
        </button>
      </div>
    </div>
  )
}

const dateFormatLabels: Record<string, string> = {
  '2006-01-02': 'YYYY-MM-DD',
  '2006/01/02': 'YYYY/MM/DD',
  '01/02/2006': 'MM/DD/YYYY',
  '02/01/2006': 'DD/MM/YYYY',
  '1/2/2006': 'M/D/YYYY',
  '2/1/2006': 'D/M/YYYY',
  '01-02-2006': 'MM-DD-YYYY',
  '02-01-2006': 'DD-MM-YYYY',
  'Jan 2, 2006': 'Mon D, YYYY',
  '2 Jan 2006': 'D Mon YYYY',
  '02-Jan-2006': 'DD-Mon-YYYY',
  '20060102': 'YYYYMMDD',
}

function ReviewStep({
  file,
  accountId,
  preview,
  excluded,
  setExcluded,
  onMappingChange,
  onBack,
  onDone,
}: {
  file: File
  accountId: string
  preview: ImportPreview
  excluded: Set<number>
  setExcluded: (s: Set<number>) => void
  onMappingChange: (p: ImportPreview) => void
  onBack: () => void
  onDone: (b: ImportBatch) => void
}) {
  const queryClient = useQueryClient()
  const [error, setError] = useState('')

  const columnCount = Math.max(preview.headers?.length ?? 0, ...preview.sampleRows.map((r) => r.length))
  const columnLabel = (i: number) => preview.headers?.[i] || `Column ${i + 1}`

  const remap = useMutation({
    mutationFn: (mapping: ImportMapping) => previewImport(file, accountId, mapping),
    onSuccess: (p) => {
      onMappingChange(p)
      setExcluded(new Set(p.rows.filter((r) => r.status !== 'ok').map((r) => r.index)))
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Preview failed'),
  })

  const commit = useMutation({
    mutationFn: () => commitImport(file, accountId, preview.mapping, [...excluded], preview.fileSha256),
    onSuccess: async (batch) => {
      await queryClient.invalidateQueries({ queryKey: ['transactions'] })
      await queryClient.invalidateQueries({ queryKey: ['accounts'] })
      await queryClient.invalidateQueries({ queryKey: ['imports'] })
      onDone(batch)
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Import failed'),
  })

  const setMapping = (patch: Partial<ImportMapping>) => remap.mutate({ ...preview.mapping, ...patch })

  const stats = useMemo(() => {
    const importing = preview.rows.filter((r) => r.status !== 'error' && !excluded.has(r.index)).length
    const dupes = preview.rows.filter((r) => r.status === 'duplicate' || r.status === 'duplicate_in_file').length
    const errors = preview.rows.filter((r) => r.status === 'error').length
    return { importing, dupes, errors }
  }, [preview.rows, excluded])

  const m = preview.mapping
  const columnSelect = (value: number, onChange: (v: number) => void, allowNone = false) => (
    <select className={inputCls} value={value} onChange={(e) => onChange(Number(e.target.value))}>
      {allowNone && <option value={-1}>—</option>}
      {Array.from({ length: columnCount }, (_, i) => (
        <option key={i} value={i}>
          {columnLabel(i)}
        </option>
      ))}
    </select>
  )

  return (
    <div className="mb-10 space-y-6">
      <section className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-6">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          1 · Column mapping
        </h2>
        <div className="grid gap-3 sm:grid-cols-3">
          <Field label="Date">{columnSelect(m.dateColumn, (v) => setMapping({ dateColumn: v }))}</Field>
          <Field label="Date format">
            <select className={inputCls} value={m.dateFormat} onChange={(e) => setMapping({ dateFormat: e.target.value })}>
              {Object.entries(dateFormatLabels).map(([layout, label]) => (
                <option key={layout} value={layout}>
                  {label}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Payee">{columnSelect(m.payeeColumn, (v) => setMapping({ payeeColumn: v }), true)}</Field>
          <Field label="Amount style">
            <select
              className={inputCls}
              value={m.amountMode}
              onChange={(e) => setMapping({ amountMode: e.target.value as ImportMapping['amountMode'] })}
            >
              <option value="signed">Single signed column</option>
              <option value="split">Separate debit / credit</option>
            </select>
          </Field>
          {m.amountMode === 'signed' ? (
            <>
              <Field label="Amount">{columnSelect(m.amountColumn, (v) => setMapping({ amountColumn: v }))}</Field>
              <label className="flex items-end gap-2 pb-2 text-sm text-slate-700 dark:text-slate-300">
                <input
                  type="checkbox"
                  checked={m.invertSign}
                  onChange={(e) => setMapping({ invertSign: e.target.checked })}
                />
                Positive means expense (credit cards)
              </label>
            </>
          ) : (
            <>
              <Field label="Debit (money out)">{columnSelect(m.debitColumn, (v) => setMapping({ debitColumn: v }))}</Field>
              <Field label="Credit (money in)">{columnSelect(m.creditColumn, (v) => setMapping({ creditColumn: v }))}</Field>
            </>
          )}
          <Field label="Notes">{columnSelect(m.notesColumn, (v) => setMapping({ notesColumn: v }), true)}</Field>
        </div>
      </section>

      <section className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-6">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
            2 · Review {preview.rowCount} rows
          </h2>
          <div className="text-sm text-slate-500 dark:text-slate-400">
            <span className="font-medium text-emerald-600 dark:text-emerald-400">{stats.importing} to import</span>
            {stats.dupes > 0 && <span> · {stats.dupes} possible duplicates</span>}
            {stats.errors > 0 && <span className="text-red-600 dark:text-red-400"> · {stats.errors} unparseable</span>}
          </div>
        </div>

        <div className="max-h-96 overflow-y-auto rounded-lg border border-slate-100 dark:border-slate-800">
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-slate-50 dark:bg-slate-800">
              <tr className="text-left text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
                <th className="px-3 py-2 font-medium">Import</th>
                <th className="px-3 py-2 font-medium">Date</th>
                <th className="px-3 py-2 font-medium">Payee</th>
                <th className="px-3 py-2 text-right font-medium">Amount</th>
                <th className="px-3 py-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {preview.rows.map((r) => (
                <tr
                  key={r.index}
                  className={`border-t border-slate-100 dark:border-slate-800 ${
                    r.status === 'error' ? 'opacity-50' : excluded.has(r.index) ? 'opacity-60' : ''
                  }`}
                >
                  <td className="px-3 py-1.5">
                    <input
                      type="checkbox"
                      disabled={r.status === 'error'}
                      checked={r.status !== 'error' && !excluded.has(r.index)}
                      onChange={(e) => {
                        const next = new Set(excluded)
                        if (e.target.checked) next.delete(r.index)
                        else next.add(r.index)
                        setExcluded(next)
                      }}
                    />
                  </td>
                  <td className="whitespace-nowrap px-3 py-1.5 text-slate-600 dark:text-slate-300">{r.date ?? '—'}</td>
                  <td className="max-w-64 truncate px-3 py-1.5 text-slate-900 dark:text-slate-100">{r.payee}</td>
                  <td className="whitespace-nowrap px-3 py-1.5 text-right">
                    {r.status !== 'error' && <MoneyAmount cents={r.amountCents} colored />}
                  </td>
                  <td className="px-3 py-1.5 text-xs">
                    {r.status === 'ok' && <span className="text-slate-400">new</span>}
                    {r.status === 'duplicate' && (
                      <span className="text-amber-600 dark:text-amber-400" title={`Matches ${r.matched?.date} ${r.matched?.payee}`}>
                        already imported
                      </span>
                    )}
                    {r.status === 'duplicate_in_file' && (
                      <span className="text-amber-600 dark:text-amber-400">repeated in file</span>
                    )}
                    {r.status === 'error' && <span className="text-red-600 dark:text-red-400">{r.error}</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
      <div className="flex gap-3">
        <button type="button" className={secondaryBtn} onClick={onBack}>
          Start over
        </button>
        <button
          type="button"
          className={primaryBtn}
          disabled={commit.isPending || remap.isPending || stats.importing === 0}
          onClick={() => commit.mutate()}
        >
          {commit.isPending ? 'Importing…' : `Import ${stats.importing} transactions`}
        </button>
      </div>
    </div>
  )
}

function SuccessCard({ batch, onReset }: { batch: ImportBatch; onReset: () => void }) {
  return (
    <div className="mb-10 max-w-lg rounded-xl border border-emerald-200 dark:border-emerald-900 bg-emerald-50 dark:bg-emerald-950/40 p-6">
      <h2 className="text-lg font-semibold text-emerald-800 dark:text-emerald-200">Import complete</h2>
      <p className="mt-1 text-sm text-emerald-700 dark:text-emerald-300">
        Imported {batch.importedCount} of {batch.rowCount} rows from {batch.filename}
        {batch.skippedCount > 0 && ` (${batch.skippedCount} skipped)`}. You can revert this import below at any time.
      </p>
      <button type="button" className={`mt-4 ${primaryBtn}`} onClick={onReset}>
        Import another file
      </button>
    </div>
  )
}

function ImportHistory() {
  const queryClient = useQueryClient()
  const { data: imports } = useQuery({ queryKey: ['imports'], queryFn: fetchImports })
  const revert = useMutation({
    mutationFn: revertImport,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['imports'] })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
    },
  })

  if (!imports || imports.length === 0) return null

  return (
    <section>
      <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Import history
      </h2>
      <div className="overflow-hidden rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900">
        {imports.map((b, i) => (
          <div
            key={b.id}
            className={`flex flex-wrap items-center justify-between gap-2 px-4 py-3 ${
              i > 0 ? 'border-t border-slate-100 dark:border-slate-800' : ''
            }`}
          >
            <div>
              <div className="text-sm font-medium text-slate-900 dark:text-slate-100">{b.filename}</div>
              <div className="text-xs text-slate-500 dark:text-slate-400">
                {b.accountName} · {b.importedCount} imported
                {b.skippedCount > 0 && `, ${b.skippedCount} skipped`} ·{' '}
                {new Date(b.createdAt).toLocaleString('en-CA')}
              </div>
            </div>
            <button
              type="button"
              className={dangerBtn}
              disabled={revert.isPending}
              onClick={() => {
                if (window.confirm(`Revert this import? Its ${b.importedCount} transactions will be deleted.`))
                  revert.mutate(b.id)
              }}
            >
              Revert
            </button>
          </div>
        ))}
      </div>
    </section>
  )
}
