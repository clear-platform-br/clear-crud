import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PublicDefinition } from '@clear-platform-br/crud-client'
import CrudFilters from './CrudFilters.vue'
import { createTranslator, ptBR } from '../messages.js'

const definition: PublicDefinition = {
  Key: 'tickets', Labels: { Title: 'tickets.title', Singular: 'tickets.singular' },
  Fields: [
    { Key: 'name', Label: 'tickets.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false },
    { Key: 'status', Label: 'tickets.status', Type: 'enum', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Enum: [{ Value: 'new', Label: 'tickets.status.new' }, { Value: 'review', Label: 'tickets.status.review' }, { Value: 'closed', Label: 'tickets.status.closed' }] },
    { Key: 'enabled', Label: 'tickets.enabled', Type: 'boolean', Required: true, ReadOnly: false, Visible: true, Sensitive: false },
  ],
  Details: [],
  Grid: { Columns: ['name', 'status', 'enabled'], Searchable: [], Sortable: [], DefaultSort: [], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } },
  Form: { Fields: [] }, Presentation: { Collection: 'table', Density: 'comfortable' }, Actions: ['read'],
}

describe('CrudFilters', () => {
  it('derives text, enum and boolean filters from grid metadata', async () => {
    const wrapper = mount(CrudFilters, { props: { definition, messages: ptBR, translate: createTranslator(ptBR), filters: [], loading: false } })
    await wrapper.get('[data-clear-crud-filter="name"] input').setValue('review')
    await wrapper.get('[data-clear-crud-filter="status"] input[value="review"]').setValue(true)
    await wrapper.get('[data-clear-crud-filter="enabled"] input[value="false"]').setValue(true)
    await wrapper.get('[data-clear-crud-action="apply-filters"]').trigger('click')
    expect(wrapper.emitted('apply')).toEqual([[ [
      { field: 'name', operator: 'contains', value: 'review' },
      { field: 'status', operator: 'eq', value: 'review' },
      { field: 'enabled', operator: 'eq', value: false },
    ] ]])
  })

  it('restores active values and clears them without definition changes', async () => {
    const wrapper = mount(CrudFilters, { props: { definition, messages: ptBR, translate: createTranslator(ptBR), filters: [{ field: 'status', operator: 'eq', value: 'review' }], loading: false } })
    expect((wrapper.get('[data-clear-crud-filter="status"] input[value="review"]').element as HTMLInputElement).checked).toBe(true)
    await wrapper.get('[data-clear-crud-action="clear-filters"]').trigger('click')
    expect(wrapper.emitted('apply')).toEqual([[[]]])
  })

  it('sends an IN filter when multiple enum options are checked', async () => {
    const wrapper = mount(CrudFilters, { props: { definition, messages: ptBR, translate: createTranslator(ptBR), filters: [], loading: false } })
    await wrapper.get('[data-clear-crud-filter="status"] input[value="review"]').setValue(true)
    await wrapper.get('[data-clear-crud-filter="status"] input[value="closed"]').setValue(true)
    await wrapper.get('[data-clear-crud-action="apply-filters"]').trigger('click')
    expect(wrapper.emitted('apply')).toEqual([[ [{ field: 'status', operator: 'in', values: ['review', 'closed'] }] ]])
  })

  it('removes the boolean filter when both states are selected', async () => {
    const wrapper = mount(CrudFilters, { props: { definition, messages: ptBR, translate: createTranslator(ptBR), filters: [], loading: false } })
    await wrapper.get('[data-clear-crud-filter="enabled"] input[value="true"]').setValue(true)
    await wrapper.get('[data-clear-crud-filter="enabled"] input[value="false"]').setValue(true)
    await wrapper.get('[data-clear-crud-action="apply-filters"]').trigger('click')
    expect(wrapper.emitted('apply')).toEqual([[[]]])
  })
})
