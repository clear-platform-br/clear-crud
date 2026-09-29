import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Page } from '@clear-platform-br/crud-client'
import CrudPagination from './CrudPagination.vue'
import { ptBR } from '../messages.js'

function page(current: number, total?: number, recordCount = 1): Page {
  return { records: Array.from({ length: recordCount }, (_, index) => ({ ID: `${current}-${index}`, Version: 1, Fields: {} })), page: current, size: 10, total }
}

describe('CrudPagination', () => {
  it('shows nearby pages with first and last shortcuts', () => {
    const wrapper = mount(CrudPagination, { props: { page: page(8, 1000), messages: ptBR, loading: false } })
    expect(wrapper.findAll('.crud-pagination-page').map((item) => item.text())).toEqual(['1', '7', '8', '9', '100'])
    expect(wrapper.findAll('.crud-pagination-ellipsis')).toHaveLength(2)
    expect(wrapper.get('[aria-current="page"]').text()).toBe('8')
  })

  it('emits a direct page navigation without reloading the current page', async () => {
    const wrapper = mount(CrudPagination, { props: { page: page(8, 1000), messages: ptBR, loading: false } })
    await wrapper.get('.crud-pagination-page:nth-of-type(1)').trigger('click')
    await wrapper.get('[aria-current="page"]').trigger('click')
    expect(wrapper.emitted('goTo')).toEqual([[1]])
  })

  it('keeps arrow navigation when the total is not available', async () => {
    const wrapper = mount(CrudPagination, { props: { page: page(2, undefined, 10), messages: ptBR, loading: false } })
    expect(wrapper.findAll('.crud-pagination-page')).toHaveLength(0)
    await wrapper.findAll('.crud-pagination-control')[0].trigger('click')
    await wrapper.findAll('.crud-pagination-control')[1].trigger('click')
    expect(wrapper.emitted('previous')).toHaveLength(1)
    expect(wrapper.emitted('next')).toHaveLength(1)
  })
})
