import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CrudEditor from './CrudEditor.vue'
import { ptBR, createTranslator } from '../messages.js'
import type { EditorDraft, PublicDefinition } from '@clear-platform-br/crud-client'

const definition: PublicDefinition = {
  Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular' },
  Fields: [{ Key: 'name', Label: 'contacts.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false }],
  Details: [{ Key: 'destinations', Resource: 'contact_destinations', Labels: { Title: 'destinations.title', Singular: 'destinations.singular' }, Fields: [{ Key: 'address', Label: 'destinations.address', Type: 'email', Required: true, ReadOnly: false, Visible: true, Sensitive: false }], Minimum: 0, Maximum: 2, AllowCreate: true, AllowUpdate: true, AllowDelete: true }],
  List: { Columns: ['name'], Searchable: ['name'], Sortable: ['name'], DefaultSort: [{ Field: 'name', Direction: 'asc' }], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } },
  Presentation: { Collection: 'table', Density: 'comfortable' }, Actions: ['create', 'read'],
}
const editor: EditorDraft = { fields: { name: 'Ana' }, details: { destinations: [] } }

describe('CrudEditor', () => {
  it('renders a declared child collection and emits its add action', async () => {
    const wrapper = mount(CrudEditor, { props: { definition, editor, messages: ptBR, translate: createTranslator(ptBR, (key) => ({ 'destinations.title': 'Destinos' }[key])), submitting: false, lookup: async () => [] } })
    expect(wrapper.get('[data-clear-crud-detail="destinations"]').text()).toContain('Destinos')
    await wrapper.get('[data-clear-crud-detail="destinations"] .crud-action').trigger('click')
    expect(wrapper.emitted('addDetail')).toEqual([['destinations']])
  })

  it('shows a qualified server error beside the child field', async () => {
    const draft: EditorDraft = { fields: { name: 'Ana' }, details: { destinations: [{ fields: { address: 'bad' } }] } }
    const wrapper = mount(CrudEditor, { props: { definition, editor: draft, feedback: { kind: 'error', message: 'crud.ui.validation', fields: { 'destinations.address': 'crud.field.invalid' } }, messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup: async () => [] } })
    expect(wrapper.get('[data-clear-crud-detail="destinations"] .crud-field-error').text()).toBe('Valor inválido.')
  })

  it('disables a create-only field while editing', () => {
    const editingDefinition: PublicDefinition = { ...definition, Fields: [...definition.Fields, { Key: 'apartment', Label: 'contacts.apartment', Type: 'string', Required: true, ReadOnly: false, CreateOnly: true, Visible: true, Sensitive: false }] }
    const editing: EditorDraft = { id: '1', version: 1, fields: { name: 'Ana', apartment: '64' }, details: { destinations: [] } }
    const wrapper = mount(CrudEditor, { props: { definition: editingDefinition, editor: editing, messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup: async () => [] } })
    expect(wrapper.get('[data-clear-crud-field="apartment"] input').attributes('disabled')).toBeDefined()
  })
})
