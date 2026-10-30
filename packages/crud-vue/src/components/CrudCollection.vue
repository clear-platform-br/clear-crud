<script setup lang="ts">
import { computed, shallowRef, useSlots, watch } from 'vue'
import type { CrudRecord, Field, LookupOption, PublicDefinition, Value } from '@clear-platform-br/crud-client'
import type { CrudMessages, Translate } from '../messages.js'
import type { ReportSummary } from '@clear-platform-br/crud-client'
import { projectReportRows } from '../report-rows.js'
import { formatNumericValue } from '../format-number.js'

type LookupLoader = (resource: string, field: string, search: string, dependencies?: Record<string, Value>) => Promise<LookupOption[]>
const props = defineProps<{ definition: PublicDefinition; records: CrudRecord[]; report?: ReportSummary; messages: CrudMessages; translate: Translate; loading: boolean; lookup?: LookupLoader; recordDetailsLabel?: string }>()
const emit = defineEmits<{ edit: [record: CrudRecord]; remove: [record: CrudRecord]; recordDetails: [recordId: string] }>()
const slots = useSlots()
const expandedDetailsID = shallowRef('')
const hasInlineDetails = computed(() => Boolean(slots['record-details']))
const reportGroupFields = computed(() => new Set(props.definition.Grid.Report?.Groups.map((group) => group.Field) ?? []))
const columns = computed(() => props.definition.Grid.Columns.filter((key) => !reportGroupFields.value.has(key)).map((key) => props.definition.Fields.find((field) => field.Key === key)).filter((field): field is PublicDefinition['Fields'][number] => Boolean(field)))
const tableRows = computed(() => projectReportRows(props.definition.Grid.Report, props.records, props.report))
watch(() => props.records.map((record) => record.ID), (recordIDs) => {
  if (expandedDetailsID.value && !recordIDs.includes(expandedDetailsID.value)) expandedDetailsID.value = ''
})
function detailsRowID(recordID: string): string { return `crud-record-details-${encodeURIComponent(recordID)}` }
function toggleRecordDetails(recordID: string): void {
  if (hasInlineDetails.value) expandedDetailsID.value = expandedDetailsID.value === recordID ? '' : recordID
  emit('recordDetails', recordID)
}
const labelState = shallowRef<Record<string, string>>({})
const pendingLabels = shallowRef<Record<string, boolean>>({})
let lookupRequest = 0
function booleanValue(value: Value | undefined): boolean | undefined {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value === 1 ? true : value === 0 ? false : undefined
  if (typeof value === 'string') return value === '1' || value.toLowerCase() === 'true' ? true : value === '0' || value.toLowerCase() === 'false' ? false : undefined
  return undefined
}
function alignmentClass(field: Field): string {
  if (field.Type === 'boolean') return 'crud-align-boolean'
  if (field.Type === 'integer' || field.Type === 'decimal') return 'crud-align-number'
  return 'crud-align-text'
}
function booleanSymbol(field: Field, value: Value | undefined): string {
  const state = booleanValue(value)
  if (state === undefined) return '—'
  return state ? field.BooleanDisplay?.True ?? '●' : field.BooleanDisplay?.False ?? '●'
}
function booleanLabel(value: Value | undefined, messages: CrudMessages): string {
  const state = booleanValue(value)
  if (state === true) return messages.booleanTrue ?? 'Sim'
  if (state === false) return messages.booleanFalse ?? 'Não'
  return '—'
}
function sameValue(left: Value | undefined, right: Value | undefined): boolean {
  return left === right
}
function lookupValueKey(field: Field, value: Value): string {
  return `${field.Key}|${typeof value}:${String(value)}`
}
function lookupDependencies(field: Field, record: CrudRecord): Record<string, Value> {
  const dependencies = field.Lookup?.Dependencies ?? []
  return Object.fromEntries(dependencies.filter((key) => {
    const value = record.Fields[key]
    return value !== null && value !== undefined && value !== ''
  }).map((key) => [key, record.Fields[key]]))
}
async function hydrateLookupLabels() {
  if (!props.lookup) return
  const request = ++lookupRequest
  pendingLabels.value = {}
  const pending = new Set<string>()
  const jobs: Promise<void>[] = []
  for (const field of columns.value) {
    if (field.Type !== 'lookup' || !field.Lookup) continue
    for (const record of props.records) {
      const value = record.Fields[field.Key]
      if (value === null || value === undefined || value === '') continue
      const key = lookupValueKey(field, value)
      if (labelState.value[key] !== undefined || pending.has(key)) continue
      pending.add(key)
      pendingLabels.value = { ...pendingLabels.value, [key]: true }
      const dependencies = lookupDependencies(field, record)
      jobs.push((async () => {
        try {
          let options = await props.lookup?.(props.definition.Key, field.Key, '', dependencies) ?? []
          let option = options.find((item) => sameValue(item.value, value) || String(item.value) === String(value))
          // An empty lookup page is intentionally bounded. Ask for an exact
          // value only when the selected ID is outside that first page.
          if (!option) {
            options = await props.lookup?.(props.definition.Key, field.Key, String(value), dependencies) ?? []
            option = options.find((item) => sameValue(item.value, value) || String(item.value) === String(value))
          }
          if (request === lookupRequest && option) labelState.value = { ...labelState.value, [key]: option.label }
        } catch {
          // An unavailable label never leaks the storage key into the grid.
        } finally {
          if (request === lookupRequest) pendingLabels.value = { ...pendingLabels.value, [key]: false }
        }
      })())
    }
  }
  await Promise.all(jobs)
}
watch(() => [props.records, props.definition.Grid.Columns], () => { void hydrateLookupLabels() }, { immediate: true, deep: true })
function display(field: Field, value: Value | undefined, translate: Translate): string {
  if (field.Type === 'boolean') return booleanSymbol(field, value)
  if (field.Type === 'enum') {
    const option = field.Enum?.find((item) => sameValue(item.Value, value))
    if (option) return translate(option.Label)
  }
  if (field.Type === 'lookup' && value !== null && value !== undefined && value !== '') {
    const key = lookupValueKey(field, value)
    return labelState.value[key] ?? (pendingLabels.value[key] ? '…' : '—')
  }
  if (value === null || value === undefined) return '—'
  const formatted = formatNumericValue(value, field.NumericDisplay, field.Type === 'integer')
  if (formatted !== undefined) return formatted
  return String(value)
}
function groupField(key: string): Field | undefined { return props.definition.Fields.find((field) => field.Key === key) }
function displayMeasure(key: string, value: string | null): string {
  if (value === null) return '—'
  const display = props.definition.Grid.Report?.Measures.find((measure) => measure.Key === key)?.NumericDisplay
  return formatNumericValue(value, display, true) ?? value
}
</script>

<template>
  <section class="crud-collection" data-clear-crud-part="collection" :aria-busy="loading">
    <p v-if="!records.length && !loading" class="crud-empty" data-clear-crud-part="empty">{{ messages.noRecords }}</p>
    <div v-else class="crud-table-wrap">
      <table class="crud-table" data-clear-crud-part="table">
        <thead class="crud-table-head"><tr class="crud-table-row"><th v-for="field in columns" :key="field.Key" :class="['crud-table-heading', alignmentClass(field)]" scope="col">{{ translate(field.Label) }}</th><th class="crud-table-heading crud-table-actions-heading" scope="col"><span class="crud-visually-hidden">{{ messages.edit }}</span></th></tr></thead>
        <tbody class="crud-table-body">
          <template v-for="row in tableRows" :key="row.key">
            <tr :class="row.kind === 'group' ? ['crud-table-row', 'crud-report-group-row', `crud-report-group-level-${row.level}`] : ['crud-table-row', { 'crud-table-row-archived': row.record.Archived }]" :data-clear-crud-report-level="row.kind === 'group' ? row.level : undefined" :data-clear-crud-archived="row.kind === 'record' && row.record.Archived ? 'true' : undefined">
              <th v-if="row.kind === 'group'" class="crud-report-group-heading" :colspan="columns.length + 1" scope="rowgroup">
                <span class="crud-report-group-label">{{ groupField(row.field) ? translate(groupField(row.field)!.Label) : row.field }}: {{ groupField(row.field) ? display(groupField(row.field)!, row.value, translate) : row.value ?? '—' }}</span>
                <span class="crud-report-measures"><span v-for="measure in row.subtotal.Measures" :key="measure.Key" class="crud-report-measure"><span>{{ translate(definition.Grid.Report?.Measures.find((item) => item.Key === measure.Key)?.Label ?? measure.Key) }}:</span> <span class="crud-report-measure-value">{{ displayMeasure(measure.Key, measure.Value) }}</span></span></span>
              </th>
              <template v-else>
                <td v-for="field in columns" :key="field.Key" :class="['crud-table-cell', alignmentClass(field)]" :data-clear-crud-field="field.Key" :data-clear-crud-label="translate(field.Label)">
                  <span v-if="field.Type === 'boolean'" class="crud-boolean-value" :data-clear-crud-boolean="booleanValue(row.record.Fields[field.Key]) === undefined ? 'unknown' : booleanValue(row.record.Fields[field.Key]) ? 'true' : 'false'" role="img" :aria-label="booleanLabel(row.record.Fields[field.Key], messages)">{{ display(field, row.record.Fields[field.Key], translate) }}</span>
                  <span v-else>{{ display(field, row.record.Fields[field.Key], translate) }}</span>
                </td>
                <td class="crud-table-cell crud-row-actions">
                  <button v-if="recordDetailsLabel" class="crud-action crud-record-details-action" data-clear-crud-action="record-details" type="button" :aria-expanded="hasInlineDetails ? expandedDetailsID === row.record.ID : undefined" :aria-controls="hasInlineDetails ? detailsRowID(row.record.ID) : undefined" @click="toggleRecordDetails(row.record.ID)">{{ recordDetailsLabel }}</button>
                  <span v-if="row.record.Archived" class="crud-archived-badge" data-clear-crud-archived-label role="status" :aria-label="messages.archived ?? 'Deleted'">
                    <span class="crud-archived-badge-mark" aria-hidden="true">×</span>{{ messages.archived ?? 'Deleted' }}
                  </span>
                  <button v-if="definition.Actions.includes('update') && !row.record.Archived" class="crud-icon-action" data-clear-crud-action="edit" type="button" :aria-label="messages.edit" :title="messages.edit" @click="emit('edit', row.record)">✎</button>
                  <button v-if="definition.Actions.includes('delete') && !row.record.Archived" class="crud-icon-action crud-icon-action-destructive" data-clear-crud-action="delete" type="button" :aria-label="messages.remove" :title="messages.remove" @click="emit('remove', row.record)">×</button>
                </td>
              </template>
            </tr>
            <tr v-if="row.kind === 'record' && hasInlineDetails && expandedDetailsID === row.record.ID" :id="detailsRowID(row.record.ID)" class="crud-record-details-row" data-clear-crud-part="record-details">
              <td class="crud-table-cell crud-record-details-cell" :colspan="columns.length + 1">
                <slot name="record-details" :record-id="row.record.ID" />
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </section>
</template>
