import { describe, expect, it } from 'vitest'
import type { CrudRecord, PublicDefinition } from '@clear-platform-br/crud-client'
import { projectReportRows } from './report-rows.js'

const report: NonNullable<PublicDefinition['Grid']['Report']> = {
  Groups: [{ Field: 'year', Direction: 'asc' }, { Field: 'month', Direction: 'asc' }],
  Measures: [{ Key: 'count', Label: 'report.count', Operation: 'count' }],
}

describe('projectReportRows', () => {
  it('interleaves each server subtotal once and does not calculate measures', () => {
    const records: CrudRecord[] = [
      { ID: 'a', Version: 1, Fields: { year: 2025, month: 11 } },
      { ID: 'b', Version: 1, Fields: { year: 2025, month: 11 } },
      { ID: 'c', Version: 1, Fields: { year: 2025, month: 12 } },
    ]
    const rows = projectReportRows(report, records, { Subtotals: [
      { Level: 1, Values: [2025], Measures: [{ Key: 'count', Value: '9' }] },
      { Level: 2, Values: [2025, 11], Measures: [{ Key: 'count', Value: '5' }] },
      { Level: 2, Values: [2025, 12], Measures: [{ Key: 'count', Value: '4' }] },
    ] })
    expect(rows.map((row) => row.kind === 'group' ? `${row.level}:${row.value}` : `record:${row.record.ID}`)).toEqual([
      '1:2025', '2:11', 'record:a', 'record:b', '2:12', 'record:c',
    ])
    expect(rows[0]).toMatchObject({ kind: 'group', subtotal: { Measures: [{ Value: '9' }] } })
  })

  it('leaves standard CRUD rows unchanged when no report is declared', () => {
    const record: CrudRecord = { ID: 'a', Version: 1, Fields: { name: 'Ana' } }
    expect(projectReportRows(undefined, [record], undefined)).toEqual([{ kind: 'record', key: 'record:a', record }])
  })
})
