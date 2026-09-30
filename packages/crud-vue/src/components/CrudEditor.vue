<script setup lang="ts">
import { onBeforeUnmount, onMounted, shallowRef } from 'vue'
import { resolveDetailFields } from '@clear-platform-br/crud-client'
import type { CrudFeedback, DetailDefinition, EditorDraft, Field, PublicDefinition, Value } from '@clear-platform-br/crud-client'
import CrudDetailTable, { type DetailTableRow } from './CrudDetailTable.vue'
import CrudFieldControl from './CrudFieldControl.vue'
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

function activeRows(detail: DetailDefinition, editor: EditorDraft): DetailTableRow[] { return (editor.details[detail.Key] ?? []).map((row, index) => ({ row, index })).filter(({ row }) => !row.delete) }
function fieldsForDetail(detail: DetailDefinition, editor: EditorDraft): Field[] { return (editor.detailFields?.[detail.Key] ?? resolveDetailFields(detail, editor.fields)).filter((field) => field.Visible && !field.ReadOnly) }
function detailAvailable(detail: DetailDefinition, editor: EditorDraft): boolean { return Object.prototype.hasOwnProperty.call(editor.details, detail.Key) }
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
          <CrudFieldControl :field="field" :resource="definition.Key" :dependencies="lookupDependencies(editor.fields, field)" :model-value="editor.fields[field.Key]" :disabled="fieldDisabled(field, editor, submitting)" :required="required(field, Boolean(editor.id && field.CreateOnly))" :mobile="mobileViewport" :control-name="`parent-${field.Key}`" :messages="messages" :translate="translate" :lookup="lookup" @update="emit('updateField', field.Key, $event)" />
          <small v-if="feedback?.fields?.[field.Key]" class="crud-field-error">{{ translate(feedback.fields[field.Key]) }}</small>
        </label>
      </div>
      <CrudDetailTable v-for="detail in definition.Details.filter((item) => detailAvailable(item, editor))" :key="detail.Key" :detail="detail" :rows="activeRows(detail, editor)" :fields="fieldsForDetail(detail, editor)" :feedback="feedback?.fields" :messages="messages" :translate="translate" :submitting="submitting" :lookup="lookup" @update="(index, key, value) => emit('updateDetail', detail.Key, index, key, value)" @add="emit('addDetail', detail.Key)" @remove="emit('removeDetail', detail.Key, $event)" />
      <footer class="crud-editor-footer"><button class="crud-action" type="button" :disabled="submitting" @click="emit('cancel')">{{ messages.cancel }}</button><button class="crud-action crud-action-primary" type="submit" :disabled="submitting">{{ messages.save }}</button></footer>
    </form>
  </section>
</template>
