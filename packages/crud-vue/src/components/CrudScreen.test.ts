import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { h } from 'vue'
import type { CrudTransport, PublicDefinition } from '@clear-platform-br/crud-client'
import CrudScreen from './CrudScreen.vue'
import { ptBR } from '../messages.js'

const definition: PublicDefinition = {
  Key: 'tickets', Labels: { Title: 'tickets.title', Singular: 'tickets.singular' },
  Fields: [
    { Key: 'name', Label: 'tickets.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false },
    { Key: 'status', Label: 'tickets.status', Type: 'enum', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Enum: [{ Value: 'new', Label: 'tickets.status.new' }, { Value: 'review', Label: 'tickets.status.review' }, { Value: 'closed', Label: 'tickets.status.closed' }] },
  ],
  Details: [],
  Grid: { Columns: ['name', 'status'], Searchable: ['name'], Sortable: ['name'], DefaultSort: [], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } },
  Form: { Fields: [] }, Presentation: { Collection: 'table', Density: 'comfortable' }, Actions: ['read'],
}

function transport(list: CrudTransport['list']): CrudTransport {
  return {
    definition: async () => definition,
    list,
    get: async () => ({ ID: '1', Version: 1, Fields: { name: 'Ticket', status: 'new' } }),
    create: async () => ({ ID: '1', Version: 1, Fields: {} }),
    update: async () => ({ ID: '1', Version: 2, Fields: {} }),
    delete: async () => undefined,
    lookup: async () => [],
  }
}

describe('CrudScreen', () => {
  it('does not show loading after the definition is ready and applies multiple choices', async () => {
    const list = vi.fn(async (_resource: string, query: Parameters<CrudTransport['list']>[1]) => ({ records: [{ ID: '1', Version: 1, Fields: { name: 'Ticket', status: 'review' } }], page: query.page ?? 1, size: query.size ?? 25, total: 1 }))
    const wrapper = mount(CrudScreen, { props: { resource: 'tickets', client: transport(list), messages: ptBR } })
    await flushPromises()
    expect(wrapper.find('.crud-loading').exists()).toBe(false)
    await wrapper.get('[data-clear-crud-action="filters"]').trigger('click')
    await wrapper.get('[data-clear-crud-filter="status"] input[value="review"]').setValue(true)
    await wrapper.get('[data-clear-crud-filter="status"] input[value="closed"]').setValue(true)
    await wrapper.get('[data-clear-crud-action="apply-filters"]').trigger('click')
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith('tickets', expect.objectContaining({ filters: [{ field: 'status', operator: 'in', values: ['review', 'closed'] }], page: 1 }), expect.anything())
  })

  it('forwards the opt-in read-only record details event to the host', async () => {
    const list = vi.fn(async () => ({ records: [{ ID: 'invoice-42', Version: 1, Fields: { name: 'Documento', status: 'new' } }], page: 1, size: 25, total: 1 }))
    const api = transport(list)
    const update = vi.spyOn(api, 'update')
    const remove = vi.spyOn(api, 'delete')
    const wrapper = mount(CrudScreen, { props: { resource: 'tickets', client: api, messages: ptBR, recordDetailsLabel: 'Ver itens' } })
    await flushPromises()
    await wrapper.get('[data-clear-crud-action="record-details"]').trigger('click')

    expect(wrapper.emitted('recordDetails')).toEqual([['invoice-42']])
    expect(update).not.toHaveBeenCalled()
    expect(remove).not.toHaveBeenCalled()
  })

  it('forwards the host detail slot so its content appears under the selected row', async () => {
    const list = vi.fn(async () => ({ records: [{ ID: 'invoice-42', Version: 1, Fields: { name: 'Documento', status: 'new' } }], page: 1, size: 25, total: 1 }))
    const wrapper = mount(CrudScreen, {
      props: { resource: 'tickets', client: transport(list), messages: ptBR, recordDetailsLabel: 'Ver itens' },
      slots: {
        'record-details': ({ recordId }: { recordId: string }) => h('p', { 'data-loaded-details': recordId }, `Itens de ${recordId}`),
      },
    })
    await flushPromises()

    await wrapper.get('[data-clear-crud-action="record-details"]').trigger('click')

    const rows = wrapper.findAll('tbody > tr')
    expect(rows).toHaveLength(2)
    expect(rows[1].attributes('data-clear-crud-part')).toBe('record-details')
    expect(rows[1].text()).toContain('Itens de invoice-42')
    expect(wrapper.emitted('recordDetails')).toEqual([['invoice-42']])
  })
})
