import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PublicDefinition } from '@clear-platform-br/crud-client'
import CrudToolbar from './CrudToolbar.vue'
import { createTranslator, ptBR } from '../messages.js'

const definition: PublicDefinition = {
  Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular' }, Fields: [], Details: [],
  Grid: { Columns: [], Searchable: [], Sortable: [], DefaultSort: [], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } }, Form: { Fields: [] },
  Presentation: { Collection: 'table', Density: 'comfortable' }, Actions: ['create'],
}

describe('CrudToolbar preferences', () => {
  it('emits theme and density preference changes', async () => {
    const wrapper = mount(CrudToolbar, { props: { definition, messages: ptBR, translate: createTranslator(ptBR), loading: false, dark: false, compact: false } })
    await wrapper.get('[data-clear-crud-action="theme"]').trigger('click')
    await wrapper.get('[data-clear-crud-action="density"]').trigger('click')
    expect(wrapper.emitted('toggleTheme')).toHaveLength(1)
    expect(wrapper.emitted('toggleDensity')).toHaveLength(1)
  })

  it('keeps create compact while exposing an accessible label', async () => {
    const wrapper = mount(CrudToolbar, { props: { definition, messages: ptBR, translate: createTranslator(ptBR), loading: false, dark: false, compact: false } })
    const create = wrapper.get('[data-clear-crud-action="create"]')
    expect(create.text()).toBe('+')
    expect(create.classes()).toContain('crud-action-primary')
    expect(create.attributes('aria-label')).toBe('Incluir')
    expect(create.attributes('title')).toBe('Incluir')
  })

  it('supports mobile-friendly explicit search submission', async () => {
    const wrapper = mount(CrudToolbar, { props: { definition, messages: ptBR, translate: createTranslator(ptBR), loading: false, dark: false, compact: false } })
    await wrapper.get('[data-clear-crud-part="search"]').setValue('douglas')
    await wrapper.get('.crud-search-form').trigger('submit')
    expect(wrapper.emitted('search')).toEqual([['douglas'], ['douglas']])
  })

  it('shows the compact archived-record switch only when declared by the Grid', async () => {
    const archiveDefinition: PublicDefinition = { ...definition, Grid: { ...definition.Grid, ArchiveVisibility: 'active_and_archived' } }
    const wrapper = mount(CrudToolbar, { props: { definition: archiveDefinition, messages: ptBR, translate: createTranslator(ptBR), loading: false, dark: false, compact: false, includeArchived: false } })
    const toggle = wrapper.get('[data-clear-crud-action="archive-visibility"]')
    expect(toggle.text()).toBe('Excluídos')
    expect(toggle.attributes('role')).toBe('switch')
    expect(toggle.attributes('aria-checked')).toBe('false')
    expect(toggle.attributes('aria-label')).toBe('Incluir registros excluídos')
    await toggle.trigger('click')
    expect(wrapper.emitted('toggleArchived')).toHaveLength(1)
  })

  it('exposes automatic column filters for every grid definition', async () => {
    const filterDefinition: PublicDefinition = { ...definition, Grid: { ...definition.Grid, Columns: ['name'] } }
    const wrapper = mount(CrudToolbar, { props: { definition: filterDefinition, messages: ptBR, translate: createTranslator(ptBR), loading: false, dark: false, compact: false } })
    const toggle = wrapper.get('[data-clear-crud-action="filters"]')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    await toggle.trigger('click')
    expect(wrapper.emitted('toggleFilters')).toHaveLength(1)
  })
})
