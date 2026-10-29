import type { CrudRecord, PublicDefinition, ReportSubtotal, ReportSummary, Value } from '@clear-platform-br/crud-client'

export type ReportTableRow =
  | { kind: 'group'; key: string; level: number; field: string; value: Value; subtotal: ReportSubtotal }
  | { kind: 'record'; key: string; record: CrudRecord }

// This helper only interleaves server-computed subtotals with the current page.
// It intentionally performs no count, sum, or other aggregate calculation.
export function projectReportRows(
  definition: NonNullable<PublicDefinition['Grid']['Report']> | undefined,
  records: CrudRecord[],
  summary: ReportSummary | undefined,
): ReportTableRow[] {
  if (!definition || !summary) return records.map((record) => ({ kind: 'record', key: `record:${record.ID}`, record }))

  const subtotals = new Map(summary.Subtotals.map((subtotal) => [subtotalKey(subtotal.Level, subtotal.Values), subtotal]))
  const emitted = new Set<string>()
  const rows: ReportTableRow[] = []
  for (const record of records) {
    const values: Value[] = []
    for (const [index, group] of definition.Groups.entries()) {
      values.push(record.Fields[group.Field] ?? null)
      const level = index + 1
      const key = subtotalKey(level, values)
      if (emitted.has(key)) continue
      const subtotal = subtotals.get(key)
      if (!subtotal) continue
      emitted.add(key)
      rows.push({ kind: 'group', key: `group:${key}`, level, field: group.Field, value: values[index] ?? null, subtotal })
    }
    rows.push({ kind: 'record', key: `record:${record.ID}`, record })
  }
  return rows
}

function subtotalKey(level: number, values: Value[]): string {
  return `${level}:${JSON.stringify(values)}`
}
