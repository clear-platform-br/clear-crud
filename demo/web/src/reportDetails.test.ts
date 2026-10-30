import { describe, expect, it, vi } from 'vitest'
import type { CrudRecord, CrudTransport } from '@clear-platform-br/crud-client'
import { groupedReportDetailsResource, loadGroupedReportDetails } from './reportDetails'

describe('grouped report details host flow', () => {
  it('requests only the selected row’s related detail lines', async () => {
    const line: CrudRecord = { ID: '101', Version: 1, Fields: { report_id: 1, document_name: 'Documento A', line_number: 1, description: 'Parcela principal', amount_cents: 75 } }
    const list = vi.fn(async () => ({ records: [line], page: 1, size: 25, total: 1 }))
    const client = { list } as unknown as Pick<CrudTransport, 'list'>

    const result = await loadGroupedReportDetails(client, '1')

    expect(list).toHaveBeenCalledWith(groupedReportDetailsResource, {
      filters: [{ field: 'report_id', operator: 'eq', value: 1 }],
      page: 1,
      size: 25,
    }, undefined)
    expect(result).toEqual([line])
  })

  it('does not request data for an invalid record ID', async () => {
    const list = vi.fn()
    const client = { list } as unknown as Pick<CrudTransport, 'list'>

    await expect(loadGroupedReportDetails(client, 'not-an-id')).rejects.toThrow('positive integer')
    expect(list).not.toHaveBeenCalled()
  })
})
