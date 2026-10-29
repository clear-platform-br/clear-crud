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

  it('uses the actual integer and decimal bounds and applies an inclusive range', async () => {
    const numericDefinition: PublicDefinition = {
      ...definition,
      Fields: [...definition.Fields,
        { Key: 'year', Label: 'tickets.year', Type: 'integer', Required: false, ReadOnly: true, Visible: true, Sensitive: false },
        { Key: 'amount', Label: 'tickets.amount', Type: 'decimal', Required: false, ReadOnly: true, Visible: true, Sensitive: false },
      ],
      Grid: { ...definition.Grid, Columns: [...definition.Grid.Columns, 'year', 'amount'] },
    }
    const wrapper = mount(CrudFilters, { props: {
      definition: numericDefinition, messages: ptBR, translate: createTranslator(ptBR), filters: [], loading: false,
      filterRanges: { year: { minimum: '1969', maximum: '2003' }, amount: { minimum: '100.00', maximum: '1000000.00' } },
    } })
    const year = wrapper.findAll('[data-clear-crud-filter="year"] input')
    expect(year[0].attributes('min')).toBe('1969')
    expect(year[0].attributes('max')).toBe('2003')
    const amount = wrapper.findAll('[data-clear-crud-filter="amount"] input')
    expect(amount[0].attributes('min')).toBe('100.00')
    expect(amount[1].attributes('max')).toBe('1000000.00')
    await year[0].setValue('1970')
    await year[1].setValue('2000')
    await amount[0].setValue('100.25')
    await amount[1].setValue('500.50')
    await wrapper.get('[data-clear-crud-action="apply-filters"]').trigger('click')
    expect(wrapper.emitted('apply')?.[0]).toEqual([[
      { field: 'year', operator: 'gte', value: '1970' },
      { field: 'year', operator: 'lte', value: '2000' },
      { field: 'amount', operator: 'gte', value: '100.25' },
      { field: 'amount', operator: 'lte', value: '500.50' },
    ]])
  })

  it('prevents month values outside the available data range', async () => {
    const monthDefinition: PublicDefinition = {
      ...definition,
      Fields: [...definition.Fields, { Key: 'month', Label: 'tickets.month', Type: 'integer', Required: false, ReadOnly: true, Visible: true, Sensitive: false }],
      Grid: { ...definition.Grid, Columns: [...definition.Grid.Columns, 'month'] },
    }
    const wrapper = mount(CrudFilters, { props: {
      definition: monthDefinition, messages: ptBR, translate: createTranslator(ptBR), filters: [], loading: false,
      filterRanges: { month: { minimum: '1', maximum: '12' } },
    } })
    const month = wrapper.findAll('[data-clear-crud-filter="month"] input')
    expect(month[0].attributes('max')).toBe('12')
    await month[0].setValue('13')
    expect(wrapper.get('[data-clear-crud-action="apply-filters"]').attributes('disabled')).toBeDefined()
    await month[0].setValue('12')
    expect(wrapper.get('[data-clear-crud-action="apply-filters"]').attributes('disabled')).toBeUndefined()
  })

  it('fails closed when the adapter provides no bounds, including for stale numeric filters', () => {
    const numericDefinition: PublicDefinition = {
      ...definition,
      Fields: [...definition.Fields, { Key: 'month', Label: 'tickets.month', Type: 'integer', Required: false, ReadOnly: true, Visible: true, Sensitive: false }],
      Grid: { ...definition.Grid, Columns: [...definition.Grid.Columns, 'month'] },
    }
    const wrapper = mount(CrudFilters, { props: {
      definition: numericDefinition, messages: ptBR, translate: createTranslator(ptBR),
      filters: [{ field: 'month', operator: 'gte', value: '-7' }], loading: false,
    } })
    const month = wrapper.findAll('[data-clear-crud-filter="month"] input')
    expect(month).toHaveLength(2)
    expect(month.every((input) => input.attributes('disabled') !== undefined)).toBe(true)
    expect(wrapper.get('[data-clear-crud-filter="month"]').text()).toContain('Limites numéricos indisponíveis; este filtro está desativado.')
    expect(wrapper.get('[data-clear-crud-filter="month"]').text()).not.toContain('Use valores entre')
    expect(wrapper.get('[data-clear-crud-action="apply-filters"]').attributes('disabled')).toBeDefined()
  })
})
