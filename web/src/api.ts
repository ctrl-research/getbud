export type Me = {
  id: string
  email: string
  displayName: string
  avatarUrl: string | null
  isAdmin: boolean
}

export class ApiError extends Error {
  code: string

  constructor(code: string, message: string) {
    super(message)
    this.code = code
  }
}

async function throwApiError(res: Response): Promise<never> {
  let code = 'unknown'
  let message = `request failed (${res.status})`
  try {
    const body = await res.json()
    if (body?.error) {
      code = body.error.code ?? code
      message = body.error.message ?? message
    }
  } catch {
    // non-JSON error body; keep defaults
  }
  throw new ApiError(code, message)
}

/** Returns the signed-in user, or null when there is no session. */
export async function fetchMe(): Promise<Me | null> {
  const res = await fetch('/api/v1/me')
  if (res.status === 401) return null
  if (!res.ok) await throwApiError(res)
  return res.json()
}

export async function fetchProviders(): Promise<{ providers: string[]; oidcName: string }> {
  const res = await fetch('/auth/providers')
  if (!res.ok) await throwApiError(res)
  const body = await res.json()
  return { providers: body.providers, oidcName: body.oidcName ?? '' }
}

export async function login(email: string, password: string): Promise<void> {
  const res = await fetch('/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) await throwApiError(res)
}

export async function logout(): Promise<void> {
  const res = await fetch('/auth/logout', { method: 'POST' })
  if (!res.ok) await throwApiError(res)
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) await throwApiError(res)
  return res.json()
}

async function sendJSON<T>(method: string, url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!res.ok) await throwApiError(res)
  if (res.status === 204) return undefined as T
  return res.json()
}

// ---- accounts -----------------------------------------------------------------

export type AccountType =
  | 'chequing'
  | 'savings'
  | 'credit_card'
  | 'rrsp'
  | 'tfsa'
  | 'fhsa'
  | 'non_registered'
  | 'other'

export const accountTypeLabels: Record<AccountType, string> = {
  chequing: 'Chequing',
  savings: 'Savings',
  credit_card: 'Credit card',
  rrsp: 'RRSP',
  tfsa: 'TFSA',
  fhsa: 'FHSA',
  non_registered: 'Non-registered',
  other: 'Other',
}

export type Account = {
  id: string
  name: string
  type: AccountType
  currency: string
  institution: string
  openingBalanceCents: number
  isArchived: boolean
  isInvestment: boolean
  balanceCents: number
  createdAt: string
}

export async function fetchAccounts(): Promise<Account[]> {
  const body = await getJSON<{ accounts: Account[] }>('/api/v1/accounts')
  return body.accounts
}

export function createAccount(input: {
  name: string
  type: AccountType
  currency?: string
  institution?: string
  openingBalanceCents?: number
}): Promise<Account> {
  return sendJSON('POST', '/api/v1/accounts', input)
}

export function updateAccount(
  id: string,
  patch: Partial<Pick<Account, 'name' | 'type' | 'currency' | 'institution' | 'openingBalanceCents' | 'isArchived'>>,
): Promise<Account> {
  return sendJSON('PATCH', `/api/v1/accounts/${id}`, patch)
}

export function deleteAccount(id: string): Promise<void> {
  return sendJSON('DELETE', `/api/v1/accounts/${id}`)
}

export type Snapshot = {
  id: string
  accountId: string
  asOf: string
  balanceCents: number
}

export async function fetchSnapshots(accountId: string): Promise<Snapshot[]> {
  const body = await getJSON<{ snapshots: Snapshot[] }>(`/api/v1/accounts/${accountId}/snapshots`)
  return body.snapshots
}

export function upsertSnapshot(accountId: string, asOf: string, balanceCents: number): Promise<Snapshot> {
  return sendJSON('PUT', `/api/v1/accounts/${accountId}/snapshots`, { asOf, balanceCents })
}

export function deleteSnapshot(accountId: string, snapshotId: string): Promise<void> {
  return sendJSON('DELETE', `/api/v1/accounts/${accountId}/snapshots/${snapshotId}`)
}

// ---- categories ---------------------------------------------------------------

export type CategoryKind = 'income' | 'expense'

export type Category = {
  id: string
  name: string
  kind: CategoryKind
  color: string
  isArchived: boolean
}

export async function fetchCategories(): Promise<Category[]> {
  const body = await getJSON<{ categories: Category[] }>('/api/v1/categories')
  return body.categories
}

export function createCategory(input: { name: string; kind: CategoryKind; color?: string }): Promise<Category> {
  return sendJSON('POST', '/api/v1/categories', input)
}

export function updateCategory(
  id: string,
  patch: Partial<Pick<Category, 'name' | 'color' | 'isArchived'>>,
): Promise<Category> {
  return sendJSON('PATCH', `/api/v1/categories/${id}`, patch)
}

export function deleteCategory(id: string, reassignTo?: string): Promise<void> {
  const qs = reassignTo ? `?reassignTo=${reassignTo}` : ''
  return sendJSON('DELETE', `/api/v1/categories/${id}${qs}`)
}

// ---- transactions -------------------------------------------------------------

export type Transaction = {
  id: string
  accountId: string
  accountName?: string
  date: string
  amountCents: number
  payee: string
  notes: string
  categoryId: string | null
  categoryName?: string
  categoryKind?: CategoryKind
  categoryColor?: string
  isTransfer: boolean
  importBatchId?: string
  createdAt: string
}

export type TransactionFilters = {
  from?: string
  to?: string
  accountId?: string
  categoryId?: string
  uncategorized?: boolean
  q?: string
  limit?: number
  offset?: number
}

export async function fetchTransactions(
  filters: TransactionFilters,
): Promise<{ transactions: Transaction[]; totalCount: number }> {
  const params = new URLSearchParams()
  if (filters.from) params.set('from', filters.from)
  if (filters.to) params.set('to', filters.to)
  if (filters.accountId) params.set('accountId', filters.accountId)
  if (filters.categoryId) params.set('categoryId', filters.categoryId)
  if (filters.uncategorized) params.set('uncategorized', 'true')
  if (filters.q) params.set('q', filters.q)
  if (filters.limit) params.set('limit', String(filters.limit))
  if (filters.offset) params.set('offset', String(filters.offset))
  return getJSON(`/api/v1/transactions?${params}`)
}

export function createTransaction(input: {
  accountId: string
  date: string
  amountCents: number
  payee?: string
  notes?: string
  categoryId?: string | null
}): Promise<Transaction> {
  return sendJSON('POST', '/api/v1/transactions', input)
}

export function updateTransaction(
  id: string,
  patch: {
    date?: string
    amountCents?: number
    payee?: string
    notes?: string
    categoryId?: string
    clearCategory?: boolean
  },
): Promise<Transaction> {
  return sendJSON('PATCH', `/api/v1/transactions/${id}`, patch)
}

export function deleteTransaction(id: string): Promise<void> {
  return sendJSON('DELETE', `/api/v1/transactions/${id}`)
}

export function createTransfer(input: {
  fromAccountId: string
  toAccountId: string
  date: string
  amountCents: number
  notes?: string
}): Promise<{ transactions: Transaction[] }> {
  return sendJSON('POST', '/api/v1/transactions/transfer', input)
}

// ---- contribution room --------------------------------------------------------

export type ContributionType = {
  roomCents: number | null
  notes: string
  contributedCents: number
  withdrawnCents: number
  remainingCents: number | null
  defaultHintCents: number
}

export type ContributionRoom = {
  year: number
  types: Record<'rrsp' | 'tfsa' | 'fhsa', ContributionType>
}

export function fetchContributionRoom(year?: number): Promise<ContributionRoom> {
  const qs = year ? `?year=${year}` : ''
  return getJSON(`/api/v1/contribution-room${qs}`)
}

export function putContributionRoom(
  type: 'rrsp' | 'tfsa' | 'fhsa',
  year: number,
  roomCents: number,
  notes = '',
): Promise<void> {
  return sendJSON('PUT', `/api/v1/contribution-room/${type}/${year}`, { roomCents, notes })
}

// ---- CSV import ---------------------------------------------------------------

export type ImportMapping = {
  dateColumn: number
  dateFormat: string
  payeeColumn: number
  notesColumn: number
  amountMode: 'signed' | 'split'
  amountColumn: number
  debitColumn: number
  creditColumn: number
  invertSign: boolean
}

export type ImportRow = {
  index: number
  date?: string
  amountCents: number
  payee: string
  notes?: string
  status: 'ok' | 'duplicate' | 'duplicate_in_file' | 'error'
  error?: string
  matched?: { date: string; amountCents: number; payee: string }
}

export type ImportPreview = {
  headers: string[] | null
  sampleRows: string[][]
  mapping: ImportMapping
  fileSha256: string
  rowCount: number
  rows: ImportRow[]
}

export type ImportBatch = {
  id: string
  accountId: string
  accountName?: string
  filename: string
  rowCount: number
  importedCount: number
  skippedCount: number
  createdAt: string
}

async function postForm<T>(url: string, form: FormData): Promise<T> {
  const res = await fetch(url, { method: 'POST', body: form })
  if (!res.ok) await throwApiError(res)
  return res.json()
}

export function previewImport(file: File, accountId: string, mapping?: ImportMapping): Promise<ImportPreview> {
  const form = new FormData()
  form.set('file', file)
  form.set('accountId', accountId)
  if (mapping) form.set('mapping', JSON.stringify(mapping))
  return postForm('/api/v1/imports/preview', form)
}

export function commitImport(
  file: File,
  accountId: string,
  mapping: ImportMapping,
  excludedRows: number[],
  fileSha256: string,
): Promise<ImportBatch> {
  const form = new FormData()
  form.set('file', file)
  form.set('accountId', accountId)
  form.set('mapping', JSON.stringify(mapping))
  form.set('excludedRows', JSON.stringify(excludedRows))
  form.set('fileSha256', fileSha256)
  return postForm('/api/v1/imports', form)
}

export async function fetchImports(): Promise<ImportBatch[]> {
  const body = await getJSON<{ imports: ImportBatch[] }>('/api/v1/imports')
  return body.imports
}

export function revertImport(id: string): Promise<void> {
  return sendJSON('DELETE', `/api/v1/imports/${id}`)
}

// ---- reports ------------------------------------------------------------------

export type DateRange = { from: string; to: string }

function rangeQS(range: DateRange): string {
  return `?from=${range.from}&to=${range.to}`
}

export type CategoryFlow = {
  categoryId: string | null
  name: string
  kind: CategoryKind
  color: string
  amountCents: number
}

export type ReportSummary = {
  from: string
  to: string
  incomeCents: number
  expenseCents: number
  netCents: number
  uncategorizedCount: number
  topExpenseCategories: CategoryFlow[]
}

export function fetchReportSummary(range: DateRange): Promise<ReportSummary> {
  return getJSON(`/api/v1/reports/summary${rangeQS(range)}`)
}

export type SankeyData = {
  nodes: { name: string; color?: string }[]
  links: { source: string; target: string; valueCents: number }[]
}

export function fetchReportSankey(range: DateRange): Promise<SankeyData> {
  return getJSON(`/api/v1/reports/sankey${rangeQS(range)}`)
}

export type TrendRow = {
  month: string
  categoryId: string | null
  name: string
  kind: CategoryKind | ''
  color: string
  inflowCents: number
  outflowCents: number
}

export function fetchReportTrends(range: DateRange): Promise<{ rows: TrendRow[] }> {
  return getJSON(`/api/v1/reports/trends${rangeQS(range)}`)
}

export type NetWorthRow = {
  month: string
  type: AccountType
  totalCents: number
}

export function fetchReportNetWorth(range: DateRange): Promise<{ rows: NetWorthRow[] }> {
  return getJSON(`/api/v1/reports/net-worth${rangeQS(range)}`)
}
