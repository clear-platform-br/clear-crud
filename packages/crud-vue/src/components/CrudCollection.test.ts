import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CrudCollection from './CrudCollection.vue'
import { ptBR, createTranslator } from '../messages.js'
import type { PublicDefinition } from '@clear-platform-br/crud-client'

const definition: PublicDefinition = {
  Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular' },
  Fields: [{ Key: 'name', Label: 'contacts.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false }], Details: [],
  Grid: { Columns: ['name'], Searchable: ['name'], Sortable: ['name'], DefaultSort: [{ Field: 'name', Direction: 'asc' }], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } }, Form: { Fields: [] },
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

  it('aligns values by type and uses the declared boolean symbols', () => {
    const typedDefinition: PublicDefinition = {
      ...definition,
      Fields: [
        ...definition.Fields,
        { Key: 'enabled', Label: 'contacts.enabled', Type: 'boolean', Required: true, ReadOnly: false, Visible: true, Sensitive: false, BooleanDisplay: { True: '●', False: '○' } },
        { Key: 'count', Label: 'contacts.count', Type: 'integer', Required: true, ReadOnly: false, Visible: true, Sensitive: false },
      ],
      Grid: { ...definition.Grid, Columns: ['name', 'enabled', 'count'] },
    }
    const wrapper = mount(CrudCollection, { props: { definition: typedDefinition, records: [{ ID: '1', Version: 1, Fields: { name: 'Ana', enabled: 0, count: 12 } }], messages: ptBR, translate: createTranslator(ptBR), loading: false } })
    expect(wrapper.get('[data-clear-crud-field="name"]').classes()).toContain('crud-align-text')
    expect(wrapper.get('[data-clear-crud-field="enabled"]').classes()).toContain('crud-align-boolean')
    expect(wrapper.get('[data-clear-crud-field="enabled"]').text()).toBe('○')
    expect(wrapper.get('[data-clear-crud-field="enabled"] [role="img"]').attributes('aria-label')).toBe('Não')
    expect(wrapper.get('[data-clear-crud-field="count"]').classes()).toContain('crud-align-number')
    expect(wrapper.findAll('th.crud-table-heading')[1].classes()).toContain('crud-align-boolean')
    expect(wrapper.findAll('th.crud-table-heading')[2].classes()).toContain('crud-align-number')
  })

  it('uses circles for boolean values when no custom symbols are declared', () => {
    const defaultDefinition: PublicDefinition = {
      ...definition,
      Fields: [{ Key: 'enabled', Label: 'contacts.enabled', Type: 'boolean', Required: true, ReadOnly: false, Visible: true, Sensitive: false }],
      Grid: { ...definition.Grid, Columns: ['enabled'] },
    }
    const wrapper = mount(CrudCollection, { props: { definition: defaultDefinition, records: [{ ID: '1', Version: 1, Fields: { enabled: true } }, { ID: '2', Version: 1, Fields: { enabled: false } }], messages: ptBR, translate: createTranslator(ptBR), loading: false } })
    expect(wrapper.findAll('.crud-boolean-value').map((item) => item.text())).toEqual(['●', '●'])
  })

  it('renders the translated label for enum values', () => {
    const enumDefinition: PublicDefinition = {
      ...definition,
      Fields: [{ Key: 'status', Label: 'contacts.status', Type: 'enum', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Enum: [{ Value: 'new', Label: 'contacts.status.new' }, { Value: 'closed', Label: 'contacts.status.closed' }] }],
      Grid: { ...definition.Grid, Columns: ['status'] },
    }
    const translate = (code: string) => ({ 'contacts.status.new': 'Novo', 'contacts.status.closed': 'Encerrado' }[code] ?? code)
    const wrapper = mount(CrudCollection, { props: { definition: enumDefinition, records: [{ ID: '1', Version: 1, Fields: { status: 'new' } }], messages: ptBR, translate, loading: false } })
    expect(wrapper.get('[data-clear-crud-field="status"]').text()).toBe('Novo')
  })

  it('hydrates lookup IDs to labels in the grid', async () => {
    const lookupDefinition: PublicDefinition = {
      ...definition,
      Fields: [{ Key: 'state_id', Label: 'contacts.state', Type: 'lookup', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Lookup: { Resource: 'states', ValueField: 'id', LabelField: 'name', PageSize: 25 } }],
      Grid: { ...definition.Grid, Columns: ['state_id'] },
    }
    const lookup = vi.fn(async () => [{ value: 1, label: 'São Paulo' }])
    const wrapper = mount(CrudCollection, { props: { definition: lookupDefinition, records: [{ ID: '1', Version: 1, Fields: { state_id: 1 } }], messages: ptBR, translate: createTranslator(ptBR), loading: false, lookup } })
    await flushPromises()
    expect(wrapper.get('[data-clear-crud-field="state_id"]').text()).toBe('São Paulo')
    expect(lookup).toHaveBeenCalledWith('contacts', 'state_id', '', {})
  })

  it('keeps archived rows visible but read-only', () => {
    const record = { ID: '2', Version: 1, Fields: { name: 'Arquivado' }, Archived: true }
    const wrapper = mount(CrudCollection, { props: { definition, records: [record], messages: ptBR, translate: createTranslator(ptBR), loading: false } })
    expect(wrapper.get('tbody tr').attributes('data-clear-crud-archived')).toBe('true')
    expect(wrapper.get('[data-clear-crud-archived-label]').text()).toContain('Excluído')
    expect(wrapper.get('[data-clear-crud-archived-label]').attributes('role')).toBe('status')
    expect(wrapper.find('[data-clear-crud-action="edit"]').exists()).toBe(false)
    expect(wrapper.find('[data-clear-crud-action="delete"]').exists()).toBe(false)
  })
})
