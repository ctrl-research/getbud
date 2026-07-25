/** Cents <-> display helpers. Amounts are integer cents everywhere; parsing
 * goes through strings so floats never touch money. */

const fmtCache = new Map<string, Intl.NumberFormat>()

function formatter(currency: string): Intl.NumberFormat {
  let f = fmtCache.get(currency)
  if (!f) {
    f = new Intl.NumberFormat('en-CA', { style: 'currency', currency })
    fmtCache.set(currency, f)
  }
  return f
}

export function formatCents(cents: number, currency = 'CAD'): string {
  const sign = cents < 0 ? '-' : ''
  const abs = Math.abs(cents)
  const dollars = Math.floor(abs / 100)
  const remainder = abs % 100
  // Format via the currency formatter on a safe integer-derived value.
  return sign + formatter(currency).format(Number(`${dollars}.${String(remainder).padStart(2, '0')}`))
}

/** Parses "1,234.56", "$12", "-3.5" into cents. Returns null when invalid. */
export function parseCents(input: string): number | null {
  const s = input.replace(/[$,\s]/g, '')
  if (!/^-?\d*(\.\d{0,2})?$/.test(s) || s === '' || s === '-' || s === '.') return null
  const negative = s.startsWith('-')
  const [whole, frac = ''] = (negative ? s.slice(1) : s).split('.')
  const cents = Number(whole || '0') * 100 + Number(frac.padEnd(2, '0') || '0')
  return negative ? -cents : cents
}

/** Cents -> "1234.56" for populating <input> values. */
export function centsToInput(cents: number): string {
  const sign = cents < 0 ? '-' : ''
  const abs = Math.abs(cents)
  return `${sign}${Math.floor(abs / 100)}.${String(abs % 100).padStart(2, '0')}`
}

export function todayISO(): string {
  const d = new Date()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${day}`
}
