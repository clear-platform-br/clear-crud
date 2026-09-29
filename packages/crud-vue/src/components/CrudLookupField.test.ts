import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CrudLookupField from './CrudLookupField.vue'
import { createTranslator, ptBR } from '../messages.js'
import type { Field } from '@clear-platform-br/crud-client'

const field: Field = { Key: 'category_id', Label: 'category.label', Type: 'lookup', Required: false, ReadOnly: false, Visible: true, Sensitive: false }
const dependentField: Field = { ...field, Lookup: { Resource: 'states', ValueField: 'id', LabelField: 'name', Dependencies: ['state_id'], PageSize: 25 } }

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

  it('sends only the declared dependency values with a lookup search', async () => {
    vi.useFakeTimers()
    const lookup = vi.fn(async () => [{ value: 'cat-1', label: 'Residencial' }])
    const wrapper = mount(CrudLookupField, { props: { field, resource: 'contacts', modelValue: null, dependencies: { state_id: 1 }, disabled: false, translate: createTranslator(ptBR), lookup } })
    await wrapper.get('input').setValue('Res')
    await vi.advanceTimersByTimeAsync(250)
    expect(lookup).toHaveBeenCalledWith('contacts', 'category_id', 'Res', { state_id: 1 })
    vi.useRealTimers()
  })

  it('does not open on focus and opens the complete root lookup from the toggle', async () => {
    vi.useFakeTimers()
    const lookup = vi.fn(async () => [{ value: 1, label: 'São Paulo' }, { value: 2, label: 'Paraná' }])
    const wrapper = mount(CrudLookupField, { props: { field: { ...field, Required: true }, resource: 'contacts', modelValue: null, disabled: false, translate: createTranslator(ptBR), lookup } })
    await wrapper.get('input').trigger('focus')
    await vi.advanceTimersByTimeAsync(250)
    expect(lookup).not.toHaveBeenCalled()
    expect(wrapper.find('.crud-lookup-options').exists()).toBe(false)
    await wrapper.get('.crud-lookup-toggle').trigger('click')
    await vi.advanceTimersByTimeAsync(250)
    expect(lookup).toHaveBeenCalledWith('contacts', 'category_id', '', undefined)
    expect(wrapper.findAll('.crud-lookup-option-button').map((item) => item.text())).toEqual(['São Paulo', 'Paraná'])
    vi.useRealTimers()
  })

  it('keeps the current label visible without opening options while editing', async () => {
    vi.useFakeTimers()
    const lookup = vi.fn(async () => [{ value: 1, label: 'Paraná' }, { value: 2, label: 'São Paulo' }])
    const wrapper = mount(CrudLookupField, { props: { field, resource: 'contacts', modelValue: 1, disabled: false, translate: createTranslator(ptBR), lookup } })
    await flushPromises()
    expect(lookup).toHaveBeenCalledWith('contacts', 'category_id', '1', undefined)
    expect(wrapper.get('input').element.value).toBe('Paraná')
    expect(wrapper.find('.crud-lookup-options').exists()).toBe(false)
    await wrapper.get('.crud-lookup-toggle').trigger('click')
    await vi.advanceTimersByTimeAsync(250)
    expect(wrapper.findAll('.crud-lookup-option-button').map((item) => item.text())).toEqual(['Paraná', 'São Paulo'])
    vi.useRealTimers()
  })

  it('blocks a dependent lookup until its parent has a value', () => {
    const lookup = vi.fn(async () => [{ value: 1, label: 'São Paulo' }])
    const wrapper = mount(CrudLookupField, { props: { field: dependentField, resource: 'contacts', modelValue: null, disabled: false, translate: createTranslator(ptBR), lookup } })
    expect(wrapper.get('input').attributes('disabled')).toBeDefined()
    expect(wrapper.get('.crud-lookup-message').text()).toBe('Escolha uma opção primeiro.')
    expect(lookup).not.toHaveBeenCalled()
  })

  it('shows an empty-state message when the selected parent has no children', async () => {
    vi.useFakeTimers()
    const lookup = vi.fn(async () => [])
    const wrapper = mount(CrudLookupField, { props: { field: dependentField, resource: 'contacts', modelValue: null, dependencies: { state_id: 3 }, disabled: false, translate: createTranslator(ptBR), lookup } })
    await wrapper.get('.crud-lookup-toggle').trigger('click')
    await vi.advanceTimersByTimeAsync(250)
    expect(lookup).toHaveBeenCalledWith('contacts', 'category_id', '', { state_id: 3 })
    expect(wrapper.get('.crud-lookup-message').text()).toBe('Nenhuma opção disponível.')
    expect(wrapper.find('.crud-lookup-option-button').exists()).toBe(false)
    vi.useRealTimers()
  })

  it('explains how to search beyond the bounded initial lookup page', async () => {
    vi.useFakeTimers()
    const options = Array.from({ length: 25 }, (_, index) => ({ value: index + 1, label: `Opção ${index + 1}` }))
    const lookup = vi.fn(async () => options)
    const wrapper = mount(CrudLookupField, { props: { field: dependentField, resource: 'contacts', modelValue: null, dependencies: { state_id: 1 }, disabled: false, translate: createTranslator(ptBR), lookup } })
    await wrapper.get('.crud-lookup-toggle').trigger('click')
    await vi.advanceTimersByTimeAsync(250)
    expect(wrapper.get('.crud-lookup-search-hint').text()).toBe('A lista inicial mostra até 25 opções. Digite para buscar outras.')
    vi.useRealTimers()
  })

  it('waits for three typed characters before issuing a filtered request', async () => {
    vi.useFakeTimers()
    const lookup = vi.fn(async () => [{ value: 1, label: 'São Paulo' }])
    const wrapper = mount(CrudLookupField, { props: { field: dependentField, resource: 'contacts', modelValue: null, dependencies: { state_id: 1 }, disabled: false, translate: createTranslator(ptBR), lookup } })
    await wrapper.get('input').setValue('SP')
    await vi.advanceTimersByTimeAsync(300)
    expect(lookup).not.toHaveBeenCalled()
    expect(wrapper.get('.crud-lookup-search-minimum').text()).toBe('Digite pelo menos 3 caracteres para pesquisar.')
    await wrapper.get('input').setValue('Sao')
    await vi.advanceTimersByTimeAsync(250)
    expect(lookup).toHaveBeenCalledWith('contacts', 'category_id', 'Sao', { state_id: 1 })
    vi.useRealTimers()
  })

  it('allows a declared shorter search minimum', async () => {
    vi.useFakeTimers()
    const lookup = vi.fn(async () => [{ value: 1, label: 'São Paulo' }])
    const fieldWithMinimum: Field = { ...dependentField, Lookup: { ...dependentField.Lookup!, MinSearchLength: 2 } }
    const wrapper = mount(CrudLookupField, { props: { field: fieldWithMinimum, resource: 'contacts', modelValue: null, dependencies: { state_id: 1 }, disabled: false, translate: createTranslator(ptBR), lookup } })
    await wrapper.get('input').setValue('SP')
    await vi.advanceTimersByTimeAsync(250)
    expect(lookup).toHaveBeenCalledWith('contacts', 'category_id', 'SP', { state_id: 1 })
    vi.useRealTimers()
  })
})
