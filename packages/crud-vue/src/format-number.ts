import type { NumericDisplay, Value } from '@clear-platform-br/crud-client'

const canonicalNumber = /^([+-]?)(\d+)(?:\.(\d+))?$/
const integerValue = /^[+-]?\d+$/

/** Formats a numeric value for display without converting exact strings to Number. */
export function formatNumericValue(
  value: Value | undefined,
  display: NumericDisplay | undefined,
  integerMinorUnits = false,
): string | undefined {
  if (!display || value === null || value === undefined || typeof value === 'boolean') return undefined

  const raw = numericText(value, integerMinorUnits)
  if (raw === undefined) return undefined

  const match = raw.match(canonicalNumber)
  if (!match) return undefined
  const [, sign, rawInteger, rawFraction = ''] = match
  const scale = integerMinorUnits ? display.Scale ?? 0 : 0
  let integer = rawInteger
  let fraction = rawFraction

  if (scale > 0) {
    if (!integerValue.test(raw)) return undefined
    const digits = rawInteger.padStart(scale + 1, '0')
    integer = digits.slice(0, -scale)
    fraction = digits.slice(-scale)
  }

  try {
    const currency = display.Currency
    const defaults = currency
      ? new Intl.NumberFormat(display.Locale, { style: 'currency', currency }).resolvedOptions()
      : undefined
    const requestedMinimum = display.MinimumFractionDigits ?? (scale || defaults?.minimumFractionDigits || 0)
    const minimumFractionDigits = Math.max(requestedMinimum, fraction.length)
    const maximumFractionDigits = minimumFractionDigits
    if (maximumFractionDigits > 100) return undefined

    const options: Intl.NumberFormatOptions = {
      useGrouping: true,
      minimumFractionDigits,
      maximumFractionDigits,
      ...(currency ? { style: 'currency' as const, currency } : {}),
    }
    const formatter = new Intl.NumberFormat(display.Locale, options)
    const signedInteger = BigInt(`${sign === '-' ? '-' : ''}${integer}`)
    const zeroInteger = formatter.formatToParts(0n).find((part) => part.type === 'integer')?.value ?? '0'
    const parts = signedInteger === 0n && sign === '-'
      ? formatter.formatToParts(-1n).map((part) => part.type === 'integer' ? { ...part, value: zeroInteger } : part.type === 'group' ? { ...part, value: '' } : part)
      : formatter.formatToParts(signedInteger)
    const localizedFraction = localizeDigits(fraction.padEnd(minimumFractionDigits, '0'), display.Locale)
    const rendered = parts.map((part) => part.type === 'fraction' ? localizedFraction : part.value).join('')
    return `${display.Prefix ?? ''}${rendered}${display.Suffix ?? ''}`
  } catch {
    // Invalid locale/currency metadata must not hide or corrupt the stored value.
    return undefined
  }
}

function numericText(value: string | number, requireInteger: boolean): string | undefined {
  if (typeof value === 'number') {
    if (!Number.isFinite(value) || (requireInteger && !Number.isSafeInteger(value))) return undefined
    return String(value)
  }
  return value
}

function localizeDigits(value: string, locale: string): string {
  if (!value) return ''
  const digits = new Intl.NumberFormat(locale, { useGrouping: false, maximumFractionDigits: 0 })
  return [...value].map((digit) => digits.format(Number(digit))).join('')
}
