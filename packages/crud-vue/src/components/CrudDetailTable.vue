<script setup lang="ts">
import { onBeforeUnmount, onMounted, shallowRef, watch } from 'vue'
import type { DetailDefinition, DetailMutation, Field, LookupOption, Value } from '@clear-platform-br/crud-client'
import CrudFieldControl from './CrudFieldControl.vue'
import type { CrudMessages, Translate } from '../messages.js'

export interface DetailTableRow {
  index: number
  row: DetailMutation
}

type LookupLoader = (resource: string, field: string, search: string, dependencies?: Record<string, Value>) => Promise<LookupOption[]>

const props = defineProps<{
  detail: DetailDefinition
  rows: DetailTableRow[]
  fields: Field[]
  feedback?: Record<string, string>
  messages: CrudMessages
  translate: Translate
  submitting: boolean
  lookup: LookupLoader
}>()

const emit = defineEmits<{
  add: []
  update: [index: number, key: string, value: Value]
  remove: [index: number]
}>()

const editingIndex = shallowRef<number | null>(null)
const editingFields = shallowRef<Record<string, Value>>({})
const editingNew = shallowRef(false)
const lastEditedIndex = shallowRef<number | null>(null)
const pendingAdd = shallowRef(false)
const mobileViewport = shallowRef(false)
const labelState = shallowRef<Record<string, string>>({})
const pendingLabels = shallowRef<Record<string, boolean>>({})
let mobileMedia: MediaQueryList | undefined
let lookupRequest = 0

// 1. Responsive state: the same field controls remain usable inside a table row.
function syncMobileViewport() { mobileViewport.value = mobileMedia?.matches ?? false }
onMounted(() => {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return
  mobileMedia = window.matchMedia('(max-width: 44rem)')
  syncMobileViewport()
  mobileMedia.addEventListener?.('change', syncMobileViewport)
})
onBeforeUnmount(() => {
  mobileMedia?.removeEventListener?.('change', syncMobileViewport)
  mobileMedia = undefined
})

// 2. Presentation: table cells reuse the grid's scalar, enum, boolean and lookup rules.
function fieldLabel(field: Field): string { return field.DisplayLabel?.trim() || props.translate(field.Label) }
function sameValue(left: Value | undefined, right: Value | undefined): boolean { return left === right }
function lookupValueKey(field: Field, value: Value, row: DetailTableRow): string { return `${field.Key}|${typeof value}:${String(value)}|${JSON.stringify(lookupDependencies(field, row.row.fields))}` }
function lookupDependencies(field: Field, values: Record<string, Value>): Record<string, Value> {
  const dependencies = field.Lookup?.Dependencies ?? []
  return Object.fromEntries(dependencies.filter((key) => values[key] !== null && values[key] !== undefined && values[key] !== '').map((key) => [key, values[key]]))
}
function booleanValue(value: Value | undefined): boolean | undefined {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value === 1 ? true : value === 0 ? false : undefined
  if (typeof value === 'string') return value === '1' || value.toLowerCase() === 'true' ? true : value === '0' || value.toLowerCase() === 'false' ? false : undefined
  return undefined
}
function booleanLabel(value: Value | undefined): string {
  const state = booleanValue(value)
  if (state === true) return props.messages.booleanTrue ?? 'Sim'
  if (state === false) return props.messages.booleanFalse ?? 'Não'
  return '—'
}
function booleanSymbol(field: Field, value: Value | undefined): string {
  const state = booleanValue(value)
  if (state === undefined) return '—'
  return state ? field.BooleanDisplay?.True ?? '●' : field.BooleanDisplay?.False ?? '●'
}
function alignmentClass(field: Field): string {
  if (field.Type === 'boolean') return 'crud-align-boolean'
  if (field.Type === 'integer' || field.Type === 'decimal') return 'crud-align-number'
  return 'crud-align-text'
}
function display(field: Field, value: Value | undefined, row: DetailTableRow): string {
  if (field.Type === 'boolean') return booleanSymbol(field, value)
  if (field.Type === 'enum') {
    const option = field.Enum?.find((candidate) => sameValue(candidate.Value, value))
    if (option) return props.translate(option.Label)
  }
  if (field.Type === 'lookup' && value !== null && value !== undefined && value !== '') {
    const key = lookupValueKey(field, value, row)
    return labelState.value[key] ?? (pendingLabels.value[key] ? '…' : '—')
  }
  if (value === null || value === undefined) return '—'
  return String(value)
}
async function hydrateLookupLabels() {
  const request = ++lookupRequest
  const pending = new Set<string>()
  const jobs: Promise<void>[] = []
  pendingLabels.value = {}
  for (const field of props.fields) {
    if (field.Type !== 'lookup' || !field.Lookup) continue
    for (const row of props.rows) {
      const value = row.row.fields[field.Key]
      if (value === null || value === undefined || value === '') continue
      const key = lookupValueKey(field, value, row)
      if (labelState.value[key] !== undefined || pending.has(key)) continue
      pending.add(key)
      pendingLabels.value = { ...pendingLabels.value, [key]: true }
      const dependencies = lookupDependencies(field, row.row.fields)
      jobs.push((async () => {
        try {
          let options = await props.lookup(props.detail.Resource, field.Key, '', dependencies)
          let option = options.find((candidate) => sameValue(candidate.value, value) || String(candidate.value) === String(value))
          if (!option) {
            options = await props.lookup(props.detail.Resource, field.Key, String(value), dependencies)
            option = options.find((candidate) => sameValue(candidate.value, value) || String(candidate.value) === String(value))
          }
          if (request === lookupRequest && option) labelState.value = { ...labelState.value, [key]: option.label }
        } catch { /* keep the value cell neutral when a label is unavailable */
        } finally {
          if (request === lookupRequest) pendingLabels.value = { ...pendingLabels.value, [key]: false }
        }
      })())
    }
  }
  await Promise.all(jobs)
}

// 3. Row actions: only one child row is edited at a time; the parent owns the draft.
function beginEdit(entry: DetailTableRow) {
  if (editingIndex.value !== null) return
  editingIndex.value = entry.index
  editingFields.value = { ...entry.row.fields }
  editingNew.value = !entry.row.id
}
function updateEditingField(key: string, value: Value) { editingFields.value = { ...editingFields.value, [key]: value } }
function finishEdit() {
  const index = editingIndex.value
  if (index === null) return
  for (const field of props.fields) emit('update', index, field.Key, editingFields.value[field.Key] ?? null)
  lastEditedIndex.value = index
  clearEditing()
}
function cancelEdit() {
  const index = editingIndex.value
  if (index !== null && editingNew.value) emit('remove', index)
  clearEditing()
}
function clearEditing() {
  editingIndex.value = null
  editingFields.value = {}
  editingNew.value = false
}
function addRow() {
  if (editingIndex.value !== null || props.rows.length >= props.detail.Maximum) return
  pendingAdd.value = true
  emit('add')
}
function removeRow(index: number) {
  if (editingIndex.value === index) clearEditing()
  emit('remove', index)
}
function fieldError(field: Field): string | undefined { return props.feedback?.[`${props.detail.Key}.${field.Key}`] }

watch(() => props.rows.length, (length, previousLength) => {
  if (!pendingAdd.value || length <= previousLength) return
  pendingAdd.value = false
  const entry = props.rows[length - 1]
  if (entry) beginEdit(entry)
})
watch(() => props.feedback, (feedback) => {
  if (editingIndex.value !== null || !feedback) return
  const hasFieldError = props.fields.some((field) => feedback[`${props.detail.Key}.${field.Key}`] !== undefined)
  if (!hasFieldError) return
  const entry = props.rows.find((candidate) => candidate.index === lastEditedIndex.value) ?? props.rows[0]
  if (entry) beginEdit(entry)
}, { deep: true })
watch(() => [props.rows, props.fields], () => { void hydrateLookupLabels() }, { immediate: true, deep: true })
</script>

<template>
  <fieldset class="crud-detail" :data-clear-crud-detail="detail.Key">
    <legend class="crud-detail-title">{{ translate(detail.Labels.Title) }}</legend>
    <div class="crud-detail-table-wrap">
      <table class="crud-table crud-detail-table" data-clear-crud-part="detail-table">
        <thead class="crud-table-head">
          <tr class="crud-table-row">
            <th v-for="field in fields" :key="field.Key" :class="['crud-table-heading', alignmentClass(field)]" scope="col">{{ fieldLabel(field) }}</th>
            <th class="crud-table-heading crud-table-actions-heading" scope="col"><span class="crud-visually-hidden">{{ messages.edit }}</span></th>
          </tr>
        </thead>
        <tbody class="crud-table-body">
          <tr v-if="!rows.length" class="crud-table-row">
            <td class="crud-table-cell crud-detail-empty" :colspan="fields.length + 1">{{ messages.noRecords }}</td>
          </tr>
          <tr v-for="entry in rows" v-else :key="entry.row.id ?? `new-${entry.index}`" class="crud-table-row" :data-clear-crud-detail-index="entry.index" :data-clear-crud-detail-editing="editingIndex === entry.index ? 'true' : undefined">
            <td v-for="field in fields" :key="field.Key" :class="['crud-table-cell', alignmentClass(field)]" :data-clear-crud-field="field.Key" :data-clear-crud-label="fieldLabel(field)">
              <CrudFieldControl v-if="editingIndex === entry.index" :field="field" :resource="detail.Resource" :dependencies="lookupDependencies(field, editingFields)" :model-value="editingFields[field.Key]" :disabled="submitting" :required="field.Required && field.Type !== 'boolean'" :mobile="mobileViewport" :control-name="`${detail.Key}-${entry.index}`" :aria-label="fieldLabel(field)" :messages="messages" :translate="translate" :lookup="lookup" @update="updateEditingField(field.Key, $event)" />
              <span v-else-if="field.Type === 'boolean'" class="crud-boolean-value" :data-clear-crud-boolean="booleanValue(entry.row.fields[field.Key]) === undefined ? 'unknown' : booleanValue(entry.row.fields[field.Key]) ? 'true' : 'false'" role="img" :aria-label="booleanLabel(entry.row.fields[field.Key])">{{ display(field, entry.row.fields[field.Key], entry) }}</span>
              <span v-else>{{ display(field, entry.row.fields[field.Key], entry) }}</span>
              <small v-if="editingIndex === entry.index && fieldError(field)" class="crud-field-error">{{ translate(fieldError(field) ?? '') }}</small>
            </td>
            <td class="crud-table-cell crud-row-actions">
              <template v-if="editingIndex === entry.index">
                <button class="crud-icon-action" data-clear-crud-action="save-detail" type="button" :aria-label="messages.save" :title="messages.save" :disabled="submitting" @click="finishEdit">✓</button>
                <button class="crud-icon-action" data-clear-crud-action="cancel-detail" type="button" :aria-label="messages.cancel" :title="messages.cancel" :disabled="submitting" @click="cancelEdit">×</button>
              </template>
              <template v-else>
                <button v-if="detail.AllowUpdate" class="crud-icon-action" data-clear-crud-action="edit-detail" type="button" :aria-label="messages.edit" :title="messages.edit" :disabled="submitting || editingIndex !== null" @click="beginEdit(entry)">✎</button>
                <button v-if="detail.AllowDelete" class="crud-icon-action crud-icon-action-destructive" data-clear-crud-action="delete-detail" type="button" :aria-label="messages.removeItem" :title="messages.removeItem" :disabled="submitting || editingIndex !== null" @click="removeRow(entry.index)">×</button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <button v-if="detail.AllowCreate" class="crud-action crud-detail-add" type="button" :disabled="submitting || editingIndex !== null || rows.length >= detail.Maximum" @click="addRow"><span aria-hidden="true">+</span>{{ messages.addItem }}</button>
    <small v-if="feedback?.[detail.Key]" class="crud-field-error">{{ translate(feedback[detail.Key]) }}</small>
  </fieldset>
</template>
