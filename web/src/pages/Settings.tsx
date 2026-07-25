import { useState, useSyncExternalStore } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createCategory,
  deleteCategory,
  fetchCategories,
  updateCategory,
  type Category,
  type CategoryKind,
} from '../api'
import { getTheme, setTheme, subscribeTheme, type Theme } from '../theme'
import { CategoryDot, Field, Modal, inputCls, primaryBtn, secondaryBtn } from '../components/ui'

export function SettingsPage() {
  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-8">
      <h1 className="mb-6 text-2xl font-semibold text-slate-900 dark:text-slate-100">Settings</h1>
      <ThemeSection />
      <CategoriesSection />
    </div>
  )
}

function ThemeSection() {
  const theme = useSyncExternalStore(subscribeTheme, getTheme)
  return (
    <section className="mb-8">
      <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Theme</h2>
      <div className="flex gap-2">
        {(['light', 'dark', 'system'] as Theme[]).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTheme(t)}
            className={`rounded-lg border px-4 py-2 text-sm capitalize ${
              theme === t
                ? 'border-slate-900 bg-slate-900 text-white dark:border-slate-100 dark:bg-slate-100 dark:text-slate-900'
                : 'border-slate-300 text-slate-600 dark:border-slate-600 dark:text-slate-400'
            }`}
          >
            {t}
          </button>
        ))}
      </div>
    </section>
  )
}

function CategoriesSection() {
  const { data: categories } = useQuery({ queryKey: ['categories'], queryFn: fetchCategories })
  const [adding, setAdding] = useState<CategoryKind | null>(null)

  if (!categories) return null

  return (
    <section>
      <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Categories
      </h2>
      {(['expense', 'income'] as const).map((kind) => (
        <div key={kind} className="mb-6">
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-sm font-medium capitalize text-slate-700 dark:text-slate-300">{kind}</h3>
            <button type="button" className={secondaryBtn} onClick={() => setAdding(kind)}>
              Add
            </button>
          </div>
          <div className="overflow-hidden rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900">
            {categories
              .filter((c) => c.kind === kind)
              .map((c, i) => (
                <CategoryRow key={c.id} category={c} first={i === 0} all={categories} />
              ))}
          </div>
        </div>
      ))}
      {adding && <AddCategoryModal kind={adding} onClose={() => setAdding(null)} />}
    </section>
  )
}

function CategoryRow({ category: c, first, all }: { category: Category; first: boolean; all: Category[] }) {
  const queryClient = useQueryClient()
  const [renaming, setRenaming] = useState(false)
  const [name, setName] = useState(c.name)
  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['categories'] })
    queryClient.invalidateQueries({ queryKey: ['transactions'] })
  }

  const rename = useMutation({
    mutationFn: () => updateCategory(c.id, { name: name.trim() }),
    onSuccess: () => {
      invalidate()
      setRenaming(false)
    },
  })
  const archive = useMutation({
    mutationFn: () => updateCategory(c.id, { isArchived: !c.isArchived }),
    onSuccess: invalidate,
  })
  const setColor = useMutation({
    mutationFn: (color: string) => updateCategory(c.id, { color }),
    onSuccess: invalidate,
  })
  const remove = useMutation({
    mutationFn: (reassignTo?: string) => deleteCategory(c.id, reassignTo),
    onSuccess: invalidate,
  })

  const onDelete = () => {
    const siblings = all.filter((x) => x.kind === c.kind && x.id !== c.id && !x.isArchived)
    const answer = window.prompt(
      `Delete "${c.name}"? Its transactions become uncategorized.\n` +
        `To move them to another category first, type its exact name:\n` +
        siblings.map((s) => `  ${s.name}`).join('\n') +
        `\n\nOr leave blank to just delete.`,
    )
    if (answer === null) return
    if (answer.trim() === '') {
      remove.mutate(undefined)
      return
    }
    const target = siblings.find((s) => s.name.toLowerCase() === answer.trim().toLowerCase())
    if (!target) {
      window.alert(`No category named "${answer.trim()}" — nothing deleted.`)
      return
    }
    remove.mutate(target.id)
  }

  return (
    <div
      className={`flex items-center gap-3 px-4 py-2.5 ${first ? '' : 'border-t border-slate-100 dark:border-slate-800'} ${
        c.isArchived ? 'opacity-50' : ''
      }`}
    >
      <label className="relative inline-flex cursor-pointer">
        <CategoryDot color={c.color} />
        <input
          type="color"
          className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
          value={c.color || '#94a3b8'}
          onChange={(e) => setColor.mutate(e.target.value)}
          title="Change color"
        />
      </label>
      {renaming ? (
        <form
          className="flex flex-1 gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            rename.mutate()
          }}
        >
          <input
            className="flex-1 rounded border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1 text-sm"
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <button type="submit" className="text-sm text-teal-600">
            Save
          </button>
          <button type="button" className="text-sm text-slate-400" onClick={() => setRenaming(false)}>
            Cancel
          </button>
        </form>
      ) : (
        <>
          <span className="flex-1 text-sm text-slate-900 dark:text-slate-100">{c.name}</span>
          <button
            type="button"
            className="text-xs text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
            onClick={() => {
              setName(c.name)
              setRenaming(true)
            }}
          >
            Rename
          </button>
          <button
            type="button"
            className="text-xs text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
            onClick={() => archive.mutate()}
          >
            {c.isArchived ? 'Unarchive' : 'Archive'}
          </button>
          <button type="button" className="text-xs text-slate-400 hover:text-red-600 dark:hover:text-red-400" onClick={onDelete}>
            Delete
          </button>
        </>
      )}
    </div>
  )
}

function AddCategoryModal({ kind, onClose }: { kind: CategoryKind; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [color, setColor] = useState('#0d9488')
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => createCategory({ name: name.trim(), kind, color }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['categories'] })
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to create category'),
  })

  return (
    <Modal title={`Add ${kind} category`} onClose={onClose}>
      <form
        className="space-y-3"
        onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate()
        }}
      >
        <Field label="Name">
          <input className={inputCls} required autoFocus value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label="Color">
          <input type="color" className="mt-1 h-9 w-16 cursor-pointer" value={color} onChange={(e) => setColor(e.target.value)} />
        </Field>
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        <button type="submit" disabled={mutation.isPending} className={`w-full ${primaryBtn}`}>
          Create
        </button>
      </form>
    </Modal>
  )
}
