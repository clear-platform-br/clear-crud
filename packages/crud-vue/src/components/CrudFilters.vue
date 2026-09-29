<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue'
import type { Field, Filter, PublicDefinition, Value } from '@clear-platform-br/crud-client'
import type { CrudMessages, Translate } from '../messages.js'

const props = defineProps<{
  definition: PublicDefinition
  messages: CrudMessages
  translate: Translate
  filters: Filter[]
  loading: boolean
}>()

const emit = defineEmits<{
  apply: [filters: Filter[]]
  close: []
}>()

const draft = shallowRef<Record<string, string[]>>({})

const fields = computed(() => props.definition.Grid.Columns
  .map((key) => props.definition.Fields.find((field) => field.Key === key))
  .filter((field): field is Field => field !== undefined && !field.Sensitive))

function valueToken(value: Value | undefined): string {
  if (value === null || value === undefined) return ''
  return String(value)
}

function draftFromFilters(filters: Filter[]): Record<string, string[]> {
  return Object.fromEntries(filters.map((filter) => [filter.field, filter.operator === 'in' ? (filter.values ?? []).map(valueToken) : filter.value === undefined || filter.value === null ? [] : [valueToken(filter.value)]]))
}

watch(() => props.filters, (filters) => { draft.value = draftFromFilters(filters) }, { immediate: true, deep: true })

function updateDraft(field: Field, value: string[]): void {
  draft.value = { ...draft.value, [field.Key]: value }
}

function selectedValues(field: Field): string[] { return draft.value[field.Key] ?? [] }

function toggleChoice(field: Field, value: string, checked: boolean): void {
  const values = new Set(selectedValues(field))
  if (checked) values.add(value)
  else values.delete(value)
  draft.value = { ...draft.value, [field.Key]: [...values] }
}

function clearChoice(field: Field): void { draft.value = { ...draft.value, [field.Key]: [] } }

function inputType(field: Field): string {
  return ({ integer: 'number', decimal: 'number', date: 'date', datetime: 'datetime-local', email: 'email', phone: 'tel' } as Record<string, string>)[field.Type] ?? 'search'
}

function isChoice(field: Field): boolean {
  return field.Type === 'enum' || field.Type === 'boolean'
}

function choiceValue(field: Field, raw: string): Value {
  if (field.Type === 'boolean') return raw === 'true'
  return field.Enum?.find((option) => valueToken(option.Value) === raw)?.Value ?? raw
}

function filterFor(field: Field, rawValues: string[]): Filter | undefined {
  if (!rawValues.length || rawValues.every((raw) => raw === '')) return undefined
  const raw = rawValues[0]
  if (field.Type === 'boolean' || field.Type === 'enum' || field.Type === 'integer') {
    const optionCount = field.Type === 'boolean' ? 2 : field.Enum?.length ?? 0
    if ((field.Type === 'boolean' || field.Type === 'enum') && optionCount > 0 && rawValues.length >= optionCount) return undefined
    const values = rawValues.map((item) => field.Type === 'integer' ? Number.parseInt(item, 10) : choiceValue(field, item))
    if (values.some((value) => typeof value === 'number' && Number.isNaN(value))) return undefined
    return values.length === 1 ? { field: field.Key, operator: 'eq', value: values[0] } : { field: field.Key, operator: 'in', values }
  }
  if (field.Type === 'decimal' || field.Type === 'date' || field.Type === 'datetime') return { field: field.Key, operator: 'eq', value: raw }
  return { field: field.Key, operator: 'contains', value: raw }
}

function apply(): void {
  emit('apply', fields.value.map((field) => filterFor(field, draft.value[field.Key] ?? [])).filter((filter): filter is Filter => Boolean(filter)))
}

function clear(): void {
  draft.value = {}
  emit('apply', [])
}
</script>

<template>
  <section class="crud-filter-panel" data-clear-crud-part="filters" :aria-busy="loading">
    <header class="crud-filter-header">
      <div>
        <h2 class="crud-filter-title">{{ messages.filters ?? 'Filtros' }}</h2>
        <p class="crud-filter-help">{{ messages.filterHint ?? 'Filtre cada coluna sem alterar a definição.' }}</p>
      </div>
      <button class="crud-icon-action" data-clear-crud-action="close-filters" type="button" :aria-label="messages.close" :title="messages.close" @click="emit('close')">×</button>
    </header>
    <div class="crud-filter-fields">
      <label v-for="field in fields" :key="field.Key" class="crud-filter-field" :data-clear-crud-filter="field.Key">
        <span class="crud-filter-label">{{ translate(field.Label) }}</span>
        <div v-if="isChoice(field)" class="crud-filter-options" role="group" :aria-label="translate(field.Label)">
          <label class="crud-filter-option"><input type="checkbox" :checked="selectedValues(field).length === 0" :disabled="loading" @change="clearChoice(field)"><span>{{ messages.filterAll ?? 'Todos' }}</span></label>
          <label v-if="field.Type === 'boolean'" class="crud-filter-option"><input type="checkbox" value="true" :checked="selectedValues(field).includes('true')" :disabled="loading" @change="toggleChoice(field, 'true', ($event.target as HTMLInputElement).checked)"><span>{{ messages.booleanTrue ?? 'Sim' }}</span></label>
          <label v-if="field.Type === 'boolean'" class="crud-filter-option"><input type="checkbox" value="false" :checked="selectedValues(field).includes('false')" :disabled="loading" @change="toggleChoice(field, 'false', ($event.target as HTMLInputElement).checked)"><span>{{ messages.booleanFalse ?? 'Não' }}</span></label>
          <label v-for="option in field.Enum ?? []" v-else :key="valueToken(option.Value)" class="crud-filter-option"><input type="checkbox" :value="valueToken(option.Value)" :checked="selectedValues(field).includes(valueToken(option.Value))" :disabled="loading" @change="toggleChoice(field, valueToken(option.Value), ($event.target as HTMLInputElement).checked)"><span>{{ translate(option.Label) }}</span></label>
        </div>
        <input v-else class="crud-input" :type="inputType(field)" :value="draft[field.Key]?.[0] ?? ''" :placeholder="messages.filterAll ?? 'Todos'" :disabled="loading" @input="updateDraft(field, ($event.target as HTMLInputElement).value ? [($event.target as HTMLInputElement).value] : [])">
      </label>
    </div>
    <footer class="crud-filter-footer">
      <button class="crud-action" data-clear-crud-action="clear-filters" type="button" :disabled="loading" @click="clear">{{ messages.clearFilters ?? 'Limpar filtros' }}</button>
      <button class="crud-action crud-action-primary" data-clear-crud-action="apply-filters" type="button" :disabled="loading" @click="apply">{{ messages.applyFilters ?? 'Aplicar filtros' }}</button>
    </footer>
  </section>
</template>
