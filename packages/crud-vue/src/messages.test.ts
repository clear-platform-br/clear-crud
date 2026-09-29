import { describe, expect, it } from 'vitest'
import { createTranslator, ptBR } from './messages.js'

describe('createTranslator', () => {
  it('does not expose an invalid-request transport code to the user', () => {
    expect(createTranslator(ptBR)('crud.error.invalid_request')).toBe('Valor inválido.')
  })

  it('translates the generic pattern validation message', () => {
    expect(createTranslator(ptBR)('crud.field.pattern')).toBe('O formato informado não é permitido.')
  })

  it('does not promise archived-record recovery by default', () => {
    expect(ptBR.confirmArchiveBody).toBe('O registro deixará de aparecer nas listas usuais.')
  })
})
