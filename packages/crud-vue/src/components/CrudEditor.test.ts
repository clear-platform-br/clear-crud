import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import CrudEditor from './CrudEditor.vue'
import { ptBR, createTranslator } from '../messages.js'
import type { EditorDraft, PublicDefinition } from '@clear-platform-br/crud-client'

const definition: PublicDefinition = {
  Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular' },
  Fields: [{ Key: 'name', Label: 'contacts.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false }],
  Details: [{ Key: 'destinations', Resource: 'contact_destinations', Labels: { Title: 'destinations.title', Singular: 'destinations.singular' }, Fields: [{ Key: 'address', Label: 'destinations.address', Type: 'email', Required: true, ReadOnly: false, Visible: true, Sensitive: false }], Minimum: 0, Maximum: 2, AllowCreate: true, AllowUpdate: true, AllowDelete: true }],
  Grid: { Columns: ['name'], Searchable: ['name'], Sortable: ['name'], DefaultSort: [{ Field: 'name', Direction: 'asc' }], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } }, Form: { Fields: [] },
  Presentation: { Collection: 'table', Density: 'comfortable', TitleField: 'name' }, Actions: ['create', 'read'],
}
const editor: EditorDraft = { fields: { name: 'Ana' }, details: { destinations: [] } }

describe('CrudEditor', () => {
  it('uses the declared title field for an existing record and keeps labels for new records', () => {
    const contextual = mount(CrudEditor, { props: { definition, editor: { id: 'contact-1', version: 1, fields: { name: 'Ana' }, details: { destinations: [] } }, messages: ptBR, translate: createTranslator(ptBR, (key) => ({ 'contacts.singular': 'Contato' }[key])), submitting: false, lookup: async () => [] } })
    expect(contextual.get('.crud-editor-title').text()).toBe('Ana')

    const emptyTitle = mount(CrudEditor, { props: { definition, editor: { id: 'contact-2', version: 1, fields: { name: null }, details: { destinations: [] } }, messages: ptBR, translate: createTranslator(ptBR, (key) => ({ 'contacts.singular': 'Contato' }[key])), submitting: false, lookup: async () => [] } })
    expect(emptyTitle.get('.crud-editor-title').text()).toBe('Contato')

    const creating = mount(CrudEditor, { props: { definition, editor, messages: ptBR, translate: createTranslator(ptBR, (key) => ({ 'contacts.title': 'Contatos' }[key])), submitting: false, lookup: async () => [] } })
    expect(creating.get('.crud-editor-title').text()).toBe('Contatos')
  })

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

  it('does not require a boolean field to be checked', () => {
    const booleanDefinition: PublicDefinition = { ...definition, Fields: [{ Key: 'enabled', Label: 'contacts.enabled', Type: 'boolean', Required: true, ReadOnly: false, Visible: true, Sensitive: false }], Details: [] }
    const wrapper = mount(CrudEditor, { props: { definition: booleanDefinition, editor: { fields: { enabled: false }, details: {} }, messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup: async () => [] } })
    expect(wrapper.get('[data-clear-crud-field="enabled"] input').attributes('required')).toBeUndefined()
  })

  it('keeps editor fields in one semantic row per field', () => {
    const rowDefinition: PublicDefinition = { ...definition, Fields: [...definition.Fields, { Key: 'email', Label: 'contacts.email', Type: 'email', Required: false, ReadOnly: false, Visible: true, Sensitive: false }] }
    const wrapper = mount(CrudEditor, { attachTo: document.body, props: { definition: rowDefinition, editor: { ...editor, fields: { ...editor.fields, email: 'ana@example.com' } }, messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup: async () => [] } })
    const fields = wrapper.findAll('.crud-form-fields > .crud-field')
    expect(fields).toHaveLength(2)
    for (const field of fields) {
      expect(field.find('.crud-field-label').exists()).toBe(true)
      expect(field.find('.crud-input').exists()).toBe(true)
    }
  })

  it('uses Enter to advance fields instead of submitting the form', async () => {
    const rowDefinition: PublicDefinition = { ...definition, Fields: [...definition.Fields, { Key: 'email', Label: 'contacts.email', Type: 'email', Required: false, ReadOnly: false, Visible: true, Sensitive: false }] }
    const wrapper = mount(CrudEditor, { props: { definition: rowDefinition, editor: { ...editor, fields: { ...editor.fields, email: 'ana@example.com' } }, messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup: async () => [] } })
    const nameInput = wrapper.get('[data-clear-crud-field="name"] input')
    const emailInput = wrapper.get('[data-clear-crud-field="email"] input')
    const event = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true })
    nameInput.element.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    expect(emailInput.exists()).toBe(true)
    expect(wrapper.emitted('submit')).toBeUndefined()
    wrapper.unmount()
  })

  it('uses the declared form projection order', () => {
    const formDefinition: PublicDefinition = {
      ...definition,
      Fields: [...definition.Fields, { Key: 'email', Label: 'contacts.email', Type: 'email', Required: false, ReadOnly: false, Visible: true, Sensitive: false }],
      Form: { Fields: ['email', 'name'] },
    }
    const wrapper = mount(CrudEditor, { props: { definition: formDefinition, editor: { ...editor, fields: { ...editor.fields, email: 'ana@example.com' } }, messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup: async () => [] } })
    expect(wrapper.findAll('.crud-form-fields > .crud-field').map((field) => field.attributes('data-clear-crud-field'))).toEqual(['email', 'name'])
  })

  it('forces a native select for enums on mobile even when segmented was requested', async () => {
    const originalMatchMedia = window.matchMedia
    const mobileMatchMedia = vi.fn(() => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }) as unknown as MediaQueryList)
    Object.defineProperty(window, 'matchMedia', { configurable: true, writable: true, value: mobileMatchMedia })
    const enumDefinition: PublicDefinition = {
      ...definition,
      Fields: [{ Key: 'status', Label: 'contacts.status', Type: 'enum', EnumControl: 'segmented', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Enum: [{ Value: 'new', Label: 'contacts.status.new' }, { Value: 'closed', Label: 'contacts.status.closed' }] }],
      Details: [], Grid: { ...definition.Grid, Columns: ['status'] }, Form: { Fields: ['status'] },
    }
    const wrapper = mount(CrudEditor, { props: { definition: enumDefinition, editor: { fields: { status: null }, details: {} }, messages: ptBR, translate: createTranslator(ptBR, (key) => ({ 'contacts.status.new': 'Novo', 'contacts.status.closed': 'Encerrado' }[key])), submitting: false, lookup: async () => [] } })
    await nextTick()
    expect(wrapper.get('[data-clear-crud-field="status"] select').exists()).toBe(true)
    expect(wrapper.get('[data-clear-crud-field="status"] select').findAll('option').map((option) => option.text())).toEqual(['Selecione…', 'Novo', 'Encerrado'])
    expect(wrapper.find('[data-clear-crud-field="status"] .crud-enum-option').exists()).toBe(false)
    wrapper.unmount()
    Object.defineProperty(window, 'matchMedia', { configurable: true, value: originalMatchMedia })
  })

  it('renders radio enums as radio inputs, segmented enums as joined buttons, and buttons as separate buttons', () => {
    const enumFields = [
      { Key: 'channel', Label: 'contacts.channel', Type: 'enum' as const, EnumControl: 'radio' as const, Required: true, ReadOnly: false, Visible: true, Sensitive: false, Enum: [{ Value: 'email', Label: 'contacts.channel.email' }, { Value: 'phone', Label: 'contacts.channel.phone' }] },
      { Key: 'state', Label: 'contacts.state', Type: 'enum' as const, EnumControl: 'segmented' as const, Required: true, ReadOnly: false, Visible: true, Sensitive: false, Enum: [{ Value: 'draft', Label: 'contacts.state.draft' }, { Value: 'active', Label: 'contacts.state.active' }] },
      { Key: 'tone', Label: 'contacts.tone', Type: 'enum' as const, EnumControl: 'buttons' as const, Required: true, ReadOnly: false, Visible: true, Sensitive: false, Enum: [{ Value: 'info', Label: 'contacts.tone.info' }, { Value: 'urgent', Label: 'contacts.tone.urgent' }] },
    ]
    const enumDefinition: PublicDefinition = { ...definition, Fields: enumFields, Details: [], Grid: { ...definition.Grid, Columns: ['channel', 'state', 'tone'] }, Form: { Fields: ['channel', 'state', 'tone'] } }
    const wrapper = mount(CrudEditor, { props: { definition: enumDefinition, editor: { fields: { channel: 'email', state: 'draft', tone: 'info' }, details: {} }, messages: ptBR, translate: createTranslator(ptBR), submitting: false, lookup: async () => [] } })
    expect(wrapper.findAll('.crud-radio-input')).toHaveLength(2)
    expect(wrapper.find('[data-clear-crud-field="state"] .crud-enum--segmented')).toBeTruthy()
    expect(wrapper.find('[data-clear-crud-field="state"] .crud-enum-option')).toBeTruthy()
    expect(wrapper.find('[data-clear-crud-field="tone"] .crud-enum--buttons')).toBeTruthy()
    expect(wrapper.findAll('[data-clear-crud-field="tone"] .crud-enum-option')).toHaveLength(2)
  })
})
