import { describe, expect, it } from 'vitest'
import { formatNumericValue } from './format-number.js'

describe('formatNumericValue', () => {
  it('formats integer minor units as a declared currency without losing precision', () => {
    expect(formatNumericValue('9007199254740993', {
      Locale: 'pt-BR', Currency: 'BRL', Scale: 2,
    }, true)).toBe('R$\u00a090.071.992.547.409,93')
  })

  it('formats decimal strings in the declared locale and keeps their value exact', () => {
    expect(formatNumericValue('12345678901234567890.5', {
      Locale: 'en-US', MinimumFractionDigits: 2,
    })).toBe('12,345,678,901,234,567,890.50')
  })

  it('preserves the sign when a negative scaled value is smaller than one unit', () => {
    expect(formatNumericValue('-1', {
      Locale: 'en-US', Currency: 'USD', Scale: 2,
    }, true)).toBe('-$0.01')
  })

  it('applies explicit unit labels only in the rendered string', () => {
    expect(formatNumericValue('42', {
      Locale: 'pt-BR', Prefix: 'R$ ', MinimumFractionDigits: 2,
    })).toBe('R$ 42,00')
  })

  it('leaves invalid or unsafe values untouched by returning no formatted value', () => {
    expect(formatNumericValue('not-a-number', { Locale: 'pt-BR' })).toBeUndefined()
    expect(formatNumericValue(9007199254740992, { Locale: 'pt-BR', Scale: 2 }, true)).toBeUndefined()
  })
})
