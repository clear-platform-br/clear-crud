import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CrudLookupField from './CrudLookupField.vue'
import { createTranslator, ptBR } from '../messages.js'
import type { Field } from '@clear-platform/crud-client'

const field: Field = { Key: 'category_id', Label: 'category.label', Type: 'lookup', Required: false, ReadOnly: false, Visible: true, Sensitive: false }

describe('CrudLookupField', () => {
  it('submits only an explicit selected value, never the search text', async () => {
    vi.useFakeTimers()
    const wrapper = mount(CrudLookupField, { props: { field, resource: 'contacts', modelValue: null, disabled: false, translate: createTranslator(ptBR), lookup: async () => [{ value: 'cat-1', label: 'Residencial' }] } })
    await wrapper.get('input').setValue('Res')
    await vi.advanceTimersByTimeAsync(250)
    expect(wrapper.emitted('update')).toBeUndefined()
    await wrapper.get('.crud-lookup-option-button').trigger('click')
    expect(wrapper.emitted('update')).toEqual([['cat-1']])
    vi.useRealTimers()
  })
})
