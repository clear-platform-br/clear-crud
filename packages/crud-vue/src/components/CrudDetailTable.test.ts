import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CrudDetailTable from './CrudDetailTable.vue'
import { createTranslator, ptBR } from '../messages.js'
import type { DetailDefinition, Field } from '@clear-platform-br/crud-client'

const field: Field = { Key: 'name', Label: 'items.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false }
const detail: DetailDefinition = {
  Key: 'items', Resource: 'items', Labels: { Title: 'items.title', Singular: 'item' }, Fields: [field], Minimum: 0, Maximum: 99,
  AllowCreate: true, AllowUpdate: true, AllowDelete: true,
}

function render(rows = [{ index: 0, row: { id: 'item-1', version: 1, fields: { name: 'Energia' } } }]) {
  return mount(CrudDetailTable, { props: { detail, rows, fields: [field], messages: ptBR, translate: createTranslator(ptBR, (key) => ({ 'items.title': 'Itens', 'items.name': 'Nome' }[key])), submitting: false, lookup: async () => [] } })
}

describe('CrudDetailTable', () => {
  it('renders child records as a table and edits one row through its actions', async () => {
    const wrapper = render()
    expect(wrapper.get('.crud-detail-table th').text()).toBe('Nome')
    expect(wrapper.get('[data-clear-crud-detail-index="0"]').text()).toContain('Energia')

    await wrapper.get('[data-clear-crud-action="edit-detail"]').trigger('click')
    await wrapper.get('[data-clear-crud-detail-index="0"] input').setValue('Nova energia')
    await wrapper.get('[data-clear-crud-action="save-detail"]').trigger('click')
    expect(wrapper.emitted('update')).toEqual([[0, 'name', 'Nova energia']])

    await wrapper.get('[data-clear-crud-action="delete-detail"]').trigger('click')
    expect(wrapper.emitted('remove')).toEqual([[0]])
  })

  it('opens a newly added row for focused editing after the parent draft changes', async () => {
    const wrapper = render([])
    await wrapper.get('.crud-detail-add').trigger('click')
    expect(wrapper.emitted('add')).toEqual([[]])
    await wrapper.setProps({ rows: [{ index: 0, row: { fields: { name: null } } }] })
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-clear-crud-action="save-detail"]').exists()).toBe(true)
    expect(wrapper.get('[data-clear-crud-detail-index="0"] input').exists()).toBe(true)
  })

  it('resolves lookup values to labels in the table', async () => {
    const lookupField: Field = { Key: 'category_id', Label: 'items.category', Type: 'lookup', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Lookup: { Resource: 'categories', ValueField: 'id', LabelField: 'name', PageSize: 25 } }
    const lookup = vi.fn(async () => [{ value: 7, label: 'Energia elétrica' }])
    const wrapper = mount(CrudDetailTable, { props: { detail, rows: [{ index: 0, row: { id: 'item-1', version: 1, fields: { category_id: 7 } } }], fields: [lookupField], messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup } })
    await flushPromises()
    expect(wrapper.get('[data-clear-crud-field="category_id"]').text()).toBe('Energia elétrica')
    expect(lookup).toHaveBeenCalledWith('items', 'category_id', '', {})
  })

  it('passes row dependencies and the resolved label to an edited lookup', async () => {
    const groupField: Field = { Key: 'group_id', Label: 'items.group', Type: 'lookup', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Lookup: { Resource: 'groups', ValueField: 'id', LabelField: 'name', Dependencies: ['type'], PageSize: 25 }, DisplayLabel: 'Grupo' }
    const typeField: Field = { Key: 'type', Label: 'items.type', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false }
    const lookup = vi.fn(async () => [{ value: 1, label: 'Receitas' }])
    const wrapper = mount(CrudDetailTable, { props: { detail, rows: [{ index: 0, row: { id: 'item-1', version: 1, fields: { type: 'revenue', group_id: 1 } } }], fields: [typeField, groupField], messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup } })

    await wrapper.get('[data-clear-crud-action="edit-detail"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-clear-crud-field="group_id"] input').attributes('aria-label')).toBe('Grupo')
    expect(lookup).toHaveBeenCalledWith('items', 'group_id', '1', { type: 'revenue' })
  })

  it('reopens the last edited row when a child-field error returns', async () => {
    const wrapper = render()
    await wrapper.get('[data-clear-crud-action="edit-detail"]').trigger('click')
    await wrapper.get('[data-clear-crud-action="save-detail"]').trigger('click')
    await wrapper.setProps({ feedback: { 'items.name': 'crud.field.invalid' } })

    expect(wrapper.get('[data-clear-crud-action="save-detail"]').exists()).toBe(true)
    expect(wrapper.get('.crud-field-error').text()).toBe('Valor inválido.')
  })
})
