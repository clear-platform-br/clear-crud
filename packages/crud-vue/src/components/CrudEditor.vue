<script setup lang="ts">
import { onBeforeUnmount, onMounted, shallowRef } from 'vue'
import type { CrudFeedback, DetailDefinition, EditorDraft, Field, PublicDefinition, Value } from '@clear-platform-br/crud-client'
import CrudLookupField from './CrudLookupField.vue'
import type { CrudMessages, Translate } from '../messages.js'

defineProps<{
  definition: PublicDefinition
  editor: EditorDraft
  feedback?: CrudFeedback
  messages: CrudMessages
  translate: Translate
  submitting: boolean
  lookup: (resource: string, field: string, search: string, dependencies?: Record<string, Value>) => Promise<Array<{ value: Value; label: string }>>
}>()
const emit = defineEmits<{
  updateField: [key: string, value: Value]
  updateDetail: [detail: string, index: number, key: string, value: Value]
  addDetail: [detail: string]
  removeDetail: [detail: string, index: number]
  submit: []
  cancel: []
}>()

const mobileViewport = shallowRef(false)
let mobileMedia: MediaQueryList | undefined

function syncMobileViewport() {
  mobileViewport.value = mobileMedia?.matches ?? false
}

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

function formFields(definition: PublicDefinition): Field[] {
  const fields = new Map(definition.Fields.map((field) => [field.Key, field]))
  const keys = definition.Form.Fields.length ? definition.Form.Fields : definition.Fields.map((field) => field.Key)
  return keys.map((key) => fields.get(key)).filter((field): field is Field => field !== undefined && !field.ReadOnly)
}

function valueFromEvent(field: Field, event: Event): Value {
  const target = event.target as HTMLInputElement
  if (field.Type === 'boolean') return target.checked
  if (field.Type === 'integer') return target.value === '' ? null : Number.parseInt(target.value, 10)
  if (field.Type === 'datetime') return target.value === '' ? null : new Date(target.value).toISOString()
  if (field.Type === 'decimal') return target.value === '' ? null : target.value
  return target.value === '' ? null : target.value
}
function inputType(field: Field): string { return ({ boolean: 'checkbox', integer: 'number', decimal: 'text', date: 'date', datetime: 'datetime-local', email: 'email', phone: 'tel' } as Record<string, string>)[field.Type] ?? 'text' }
function enumControl(field: Field): 'select' | 'radio' | 'segmented' | 'buttons' {
  if (mobileViewport.value) return 'select'
  if (field.EnumControl === 'select' || field.EnumControl === 'radio' || field.EnumControl === 'segmented' || field.EnumControl === 'buttons') return field.EnumControl
  return (field.Enum?.length ?? 0) <= 4 ? 'segmented' : 'select'
}
function enumIndex(field: Field, value: Value | undefined): string {
  const index = field.Enum?.findIndex((option) => option.Value === value) ?? -1
  return index >= 0 ? String(index) : ''
}
function enumValue(field: Field, event: Event): Value {
  const index = Number((event.target as HTMLSelectElement).value)
  return field.Enum?.[index]?.Value ?? null
}
function enumGroupName(resource: string, field: Field, suffix = ''): string {
  return ['crud', resource, field.Key, suffix].filter(Boolean).join('-')
}
function displayValue(field: Field, value: Value | undefined): string | number {
  if (field.Type !== 'datetime' || typeof value !== 'string' || value === '') return typeof value === 'string' || typeof value === 'number' ? value : ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toISOString().slice(0, 16)
}
function activeRows(detail: DetailDefinition, editor: EditorDraft) { return (editor.details[detail.Key] ?? []).map((row, index) => ({ row, index })).filter(({ row }) => !row.delete) }
function fieldDisabled(field: Field, editor: EditorDraft, submitting: boolean): boolean { return submitting || Boolean(editor.id && field.CreateOnly) }
function required(field: Field, editing = false): boolean { return field.Required && field.Type !== 'boolean' && !editing }
function editorTitle(definition: PublicDefinition, editor: EditorDraft, translate: Translate): string {
  if (editor.id && definition.Presentation.TitleField) {
    const value = editor.fields[definition.Presentation.TitleField]
    if (typeof value === 'string' && value.trim()) return value
    if (typeof value === 'number') return String(value)
  }
  return translate(editor.id ? definition.Labels.Singular : definition.Labels.Title)
}
function lookupDependencies(values: Record<string, Value>, field: Field): Record<string, Value> {
  const dependencies = field.Lookup?.Dependencies ?? []
  return Object.fromEntries(dependencies.filter((key) => values[key] !== null && values[key] !== undefined && values[key] !== '').map((key) => [key, values[key]]))
}
function handleFormKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.isComposing) return
  const target = event.target as HTMLElement
  if (target.closest('button') || target.tagName === 'TEXTAREA') return
  event.preventDefault()
  const form = (target.closest('form') ?? event.currentTarget) as HTMLFormElement | null
  if (!form) return
  const controls = Array.from(form.querySelectorAll<HTMLElement>('input:not([disabled]), select:not([disabled]), button:not([disabled])'))
  const next = controls[controls.findIndex((control) => control === target || control.contains(target)) + 1]
  next?.focus()
}
</script>

<template>
  <section class="crud-editor-backdrop" data-clear-crud-part="editor">
    <form class="crud-editor" @keydown="handleFormKeydown" @submit.prevent="emit('submit')">
      <header class="crud-editor-header"><h2 class="crud-editor-title">{{ editorTitle(definition, editor, translate) }}</h2><button class="crud-icon-action" type="button" :aria-label="messages.cancel" :title="messages.cancel" @click="emit('cancel')">×</button></header>
      <p v-if="feedback?.kind !== 'success'" class="crud-error-summary" role="alert">{{ feedback ? translate(feedback.message) : '' }}</p>
      <div class="crud-form-fields">
        <label v-for="field in formFields(definition)" :key="field.Key" class="crud-field" :data-clear-crud-field="field.Key">
          <span class="crud-field-label">{{ translate(field.Label) }}<span v-if="field.Required" aria-hidden="true"> *</span><span v-if="field.Help" class="crud-field-hint" :aria-label="translate(field.Help)" :title="translate(field.Help)">?</span></span>
          <CrudLookupField v-if="field.Type === 'lookup'" :field="field" :resource="definition.Key" :dependencies="lookupDependencies(editor.fields, field)" :model-value="editor.fields[field.Key]" :disabled="fieldDisabled(field, editor, submitting)" :messages="messages" :translate="translate" :lookup="lookup" @update="emit('updateField', field.Key, $event)" />
          <div v-else-if="field.Type === 'enum'" class="crud-enum-control">
            <select v-if="enumControl(field) === 'select'" class="crud-input crud-enum-select" :value="enumIndex(field, editor.fields[field.Key])" :required="required(field)" :disabled="fieldDisabled(field, editor, submitting)" :aria-label="translate(field.Label)" @change="emit('updateField', field.Key, enumValue(field, $event))">
              <option value="" disabled>{{ messages.selectOption ?? 'Selecione…' }}</option>
              <option v-for="(option, index) in field.Enum ?? []" :key="String(option.Value)" :value="index">{{ translate(option.Label) }}</option>
            </select>
            <div v-else-if="enumControl(field) === 'radio'" class="crud-enum crud-enum--radio" role="radiogroup" :aria-label="translate(field.Label)">
              <label v-for="option in field.Enum ?? []" :key="String(option.Value)" class="crud-radio-option">
                <input class="crud-radio-input" type="radio" :name="enumGroupName(definition.Key, field)" :value="String(option.Value)" :checked="editor.fields[field.Key] === option.Value" :required="required(field)" :disabled="fieldDisabled(field, editor, submitting)" @change="emit('updateField', field.Key, option.Value)">
                <span>{{ translate(option.Label) }}</span>
              </label>
            </div>
            <div v-else class="crud-enum" :class="`crud-enum--${enumControl(field)}`" role="radiogroup" :aria-label="translate(field.Label)">
              <button v-for="option in field.Enum ?? []" :key="String(option.Value)" class="crud-enum-option" type="button" role="radio" :aria-checked="editor.fields[field.Key] === option.Value" :disabled="fieldDisabled(field, editor, submitting)" @click="emit('updateField', field.Key, option.Value)">{{ translate(option.Label) }}</button>
            </div>
          </div>
          <textarea v-else-if="field.Type === 'text'" class="crud-input" :value="displayValue(field, editor.fields[field.Key])" :required="required(field, Boolean(editor.id && field.CreateOnly))" :disabled="fieldDisabled(field, editor, submitting)" @input="emit('updateField', field.Key, valueFromEvent(field, $event))" />
          <input v-else class="crud-input" :type="inputType(field)" :checked="field.Type === 'boolean' ? Boolean(editor.fields[field.Key]) : undefined" :value="field.Type === 'boolean' ? undefined : displayValue(field, editor.fields[field.Key])" :required="required(field, Boolean(editor.id && field.CreateOnly))" :disabled="fieldDisabled(field, editor, submitting)" @input="emit('updateField', field.Key, valueFromEvent(field, $event))">
          <small v-if="feedback?.fields?.[field.Key]" class="crud-field-error">{{ translate(feedback.fields[field.Key]) }}</small>
        </label>
      </div>
      <fieldset v-for="detail in definition.Details" :key="detail.Key" class="crud-detail" :data-clear-crud-detail="detail.Key">
        <legend class="crud-detail-title">{{ translate(detail.Labels.Title) }}</legend>
        <div v-for="entry in activeRows(detail, editor)" :key="entry.row.id ?? entry.index" class="crud-detail-row">
          <label v-for="field in detail.Fields" :key="field.Key" class="crud-field">
            <span class="crud-field-label">{{ translate(field.Label) }}<span v-if="field.Help" class="crud-field-hint" :aria-label="translate(field.Help)" :title="translate(field.Help)">?</span></span>
            <CrudLookupField v-if="field.Type === 'lookup'" :field="field" :resource="detail.Resource" :dependencies="lookupDependencies(entry.row.fields, field)" :model-value="entry.row.fields[field.Key]" :disabled="submitting" :messages="messages" :translate="translate" :lookup="lookup" @update="emit('updateDetail', detail.Key, entry.index, field.Key, $event)" />
            <div v-else-if="field.Type === 'enum'" class="crud-enum-control">
              <select v-if="enumControl(field) === 'select'" class="crud-input crud-enum-select" :value="enumIndex(field, entry.row.fields[field.Key])" :required="required(field)" :disabled="submitting" :aria-label="translate(field.Label)" @change="emit('updateDetail', detail.Key, entry.index, field.Key, enumValue(field, $event))">
                <option value="" disabled>{{ messages.selectOption ?? 'Selecione…' }}</option>
                <option v-for="(option, optionIndex) in field.Enum ?? []" :key="String(option.Value)" :value="optionIndex">{{ translate(option.Label) }}</option>
              </select>
              <div v-else-if="enumControl(field) === 'radio'" class="crud-enum crud-enum--radio" role="radiogroup" :aria-label="translate(field.Label)">
                <label v-for="option in field.Enum ?? []" :key="String(option.Value)" class="crud-radio-option">
                  <input class="crud-radio-input" type="radio" :name="enumGroupName(detail.Resource, field, `${detail.Key}-${entry.index}`)" :value="String(option.Value)" :checked="entry.row.fields[field.Key] === option.Value" :required="required(field)" :disabled="submitting" @change="emit('updateDetail', detail.Key, entry.index, field.Key, option.Value)">
                  <span>{{ translate(option.Label) }}</span>
                </label>
              </div>
              <div v-else class="crud-enum" :class="`crud-enum--${enumControl(field)}`" role="radiogroup" :aria-label="translate(field.Label)">
                <button v-for="option in field.Enum ?? []" :key="String(option.Value)" class="crud-enum-option" type="button" role="radio" :aria-checked="entry.row.fields[field.Key] === option.Value" :disabled="submitting" @click="emit('updateDetail', detail.Key, entry.index, field.Key, option.Value)">{{ translate(option.Label) }}</button>
              </div>
            </div>
            <textarea v-else-if="field.Type === 'text'" class="crud-input" :value="displayValue(field, entry.row.fields[field.Key])" :required="required(field)" :disabled="submitting" @input="emit('updateDetail', detail.Key, entry.index, field.Key, valueFromEvent(field, $event))" />
            <input v-else class="crud-input" :type="inputType(field)" :checked="field.Type === 'boolean' ? Boolean(entry.row.fields[field.Key]) : undefined" :value="field.Type === 'boolean' ? undefined : displayValue(field, entry.row.fields[field.Key])" :required="required(field)" :disabled="submitting" @input="emit('updateDetail', detail.Key, entry.index, field.Key, valueFromEvent(field, $event))">
            <small v-if="feedback?.fields?.[`${detail.Key}.${field.Key}`]" class="crud-field-error">{{ translate(feedback.fields[`${detail.Key}.${field.Key}`]) }}</small>
          </label>
          <button v-if="detail.AllowDelete" class="crud-icon-action crud-icon-action-destructive" type="button" :aria-label="messages.removeItem" :title="messages.removeItem" :disabled="submitting" @click="emit('removeDetail', detail.Key, entry.index)">×</button>
        </div>
        <button v-if="detail.AllowCreate" class="crud-action" type="button" :disabled="submitting" @click="emit('addDetail', detail.Key)"><span aria-hidden="true">+</span>{{ messages.addItem }}</button>
        <small v-if="feedback?.fields?.[detail.Key]" class="crud-field-error">{{ translate(feedback.fields[detail.Key]) }}</small>
      </fieldset>
      <footer class="crud-editor-footer"><button class="crud-action" type="button" :disabled="submitting" @click="emit('cancel')">{{ messages.cancel }}</button><button class="crud-action crud-action-primary" type="submit" :disabled="submitting">{{ messages.save }}</button></footer>
    </form>
  </section>
</template>
