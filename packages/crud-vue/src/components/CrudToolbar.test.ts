import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CrudToolbar from './CrudToolbar.vue'
import { createTranslator, ptBR } from '../messages.js'
import type { PublicDefinition } from '@clear-platform/crud-client'

const definition: PublicDefinition = {
  Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular', Help: 'contacts.help' }, Fields: [], Details: [],
  List: { Columns: [], Searchable: [], Sortable: [], DefaultSort: [], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } },
  Presentation: { Collection: 'table', Density: 'comfortable' }, Actions: ['read', 'help'],
}

describe('CrudToolbar', () => {
  it('exposes help through the standard question-mark action', async () => {
    const wrapper = mount(CrudToolbar, { props: { definition, messages: ptBR, translate: createTranslator(ptBR), loading: false } })
    const help = wrapper.get('[data-clear-crud-action="help"]')
    expect(help.text()).toBe('?')
    expect(help.attributes('aria-label')).toBe('Ajuda')
    await help.trigger('click')
    expect(wrapper.emitted('help')).toEqual([[]])
  })
})
