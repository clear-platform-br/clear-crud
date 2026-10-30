import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { CrudRecord } from '@clear-platform-br/crud-client'
import ReportDetailsPanel from './ReportDetailsPanel.vue'

describe('ReportDetailsPanel', () => {
  it('keeps the dialog open while detail rows load and until the user closes it', async () => {
    const wrapper = mount(ReportDetailsPanel, {
      props: {
        open: true,
        recordId: '1',
        lines: [],
        loading: true,
        failed: false,
      },
    })

    expect(wrapper.get('[role="dialog"]').text()).toContain('Carregando linhas de detalhe')

    const lines: CrudRecord[] = [
      { ID: '101', Version: 1, Fields: { document_name: 'Documento A', line_number: 1, description: 'Parcela principal', amount_cents: 75 } },
      { ID: '102', Version: 1, Fields: { document_name: 'Documento A', line_number: 2, description: 'Taxa', amount_cents: 25 } },
    ]
    await wrapper.setProps({ lines, loading: false })

    const dialog = wrapper.get('[role="dialog"]')
    expect(dialog.text()).toContain('Detalhes de Documento A')
    expect(dialog.text()).toContain('Parcela principal')
    expect(dialog.text()).toContain('Taxa')
    expect(wrapper.emitted('close')).toBeUndefined()

    await wrapper.get('button[aria-label="Fechar detalhes"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
