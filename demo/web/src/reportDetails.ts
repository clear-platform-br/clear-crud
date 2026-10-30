import type { CrudRecord, CrudTransport } from '@clear-platform-br/crud-client'

export const groupedReportResource = 'clear_crud_grouped_report_demo'
export const groupedReportDetailsResource = 'clear_crud_grouped_report_detail_lines_demo'

/** Loads only the selected report's read-only detail lines through the host transport. */
export async function loadGroupedReportDetails(
  client: Pick<CrudTransport, 'list'>,
  recordID: string,
  signal?: AbortSignal,
): Promise<CrudRecord[]> {
  const reportID = Number(recordID)
  if (!Number.isSafeInteger(reportID) || reportID < 1) {
    throw new Error('The report record ID must be a positive integer.')
  }

  const page = await client.list(groupedReportDetailsResource, {
    filters: [{ field: 'report_id', operator: 'eq', value: reportID }],
    page: 1,
    size: 25,
  }, signal)
  return page.records
}
