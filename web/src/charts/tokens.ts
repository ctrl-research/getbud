/**
 * Chart color tokens: the validated reference palette (dataviz skill), one
 * set per mode. Categorical slot order is the CVD-safety mechanism - assign
 * hues to entities in this fixed order, never cycled, never by rank.
 */
export type ChartTokens = {
  surface: string
  textPrimary: string
  textSecondary: string
  muted: string
  gridline: string
  baseline: string
  series: string[]
}

export const lightTokens: ChartTokens = {
  surface: '#fcfcfb',
  textPrimary: '#0b0b0b',
  textSecondary: '#52514e',
  muted: '#898781',
  gridline: '#e1e0d9',
  baseline: '#c3c2b7',
  series: ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#4a3aa7', '#e34948'],
}

export const darkTokens: ChartTokens = {
  surface: '#1a1a19',
  textPrimary: '#ffffff',
  textSecondary: '#c3c2b7',
  muted: '#898781',
  gridline: '#2c2c2a',
  baseline: '#383835',
  series: ['#3987e5', '#d95926', '#199e70', '#c98500', '#d55181', '#008300', '#9085e9', '#e66767'],
}

export function isDarkMode(): boolean {
  return document.documentElement.classList.contains('dark')
}

export function currentTokens(): ChartTokens {
  return isDarkMode() ? darkTokens : lightTokens
}

/** Fixed slot per account type (identity - filters must not repaint). */
export const accountTypeSlot: Record<string, number> = {
  chequing: 0,
  savings: 1,
  tfsa: 2,
  rrsp: 3,
  fhsa: 4,
  non_registered: 5,
  credit_card: 6,
  other: 7,
}
