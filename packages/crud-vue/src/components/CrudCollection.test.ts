import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CrudCollection from './CrudCollection.vue'
import { ptBR, createTranslator } from '../messages.js'
import type { PublicDefinition } from '@clear-platform-br/crud-client'

const definition: PublicDefinition = {
  Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular' },
  Fields: [{ Key: 'name', Label: 'contacts.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false }], Details: [],
  List: { Columns: ['name'], Searchable: ['name'], Sortable: ['name'], DefaultSort: [{ Field: 'name', Direction: 'asc' }], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } },
  Presentation: { Collection: 'table', Density: 'comfortable' }, Actions: ['read', 'update', 'delete'],
}

describe('CrudCollection', () => {
  it('uses accessible symbol actions and emits the selected record', async () => {
    const record = { ID: '1', Version: 1, Fields: { name: 'Ana' } }
    const wrapper = mount(CrudCollection, { props: { definition, records: [record], messages: ptBR, translate: createTranslator(ptBR), loading: false } })
    expect(wrapper.get('[data-clear-crud-action="edit"]').attributes('aria-label')).toBe('Editar')
    await wrapper.get('[data-clear-crud-action="edit"]').trigger('click')
    await wrapper.get('[data-clear-crud-action="delete"]').trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual([record])
    expect(wrapper.emitted('remove')?.[0]).toEqual([record])
  })
})
