<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue'
import type { Field, Filter, NumericFilterRange, PublicDefinition, Value } from '@clear-platform-br/crud-client'
import type { CrudMessages, Translate } from '../messages.js'

const props = defineProps<{
  definition: PublicDefinition
  messages: CrudMessages
  translate: Translate
  filters: Filter[]
  filterRanges?: Record<string, NumericFilterRange>
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
  const result: Record<string, string[]> = {}
  for (const filter of filters) {
    const field = props.definition.Fields.find((candidate) => candidate.Key === filter.field)
    if (field && (field.Type === 'integer' || field.Type === 'decimal')) {
      const values = result[filter.field] ?? ['', '']
      const raw = filter.value === undefined || filter.value === null ? '' : valueToken(filter.value)
      if (filter.operator === 'eq') { values[0] = raw; values[1] = raw }
      else if (filter.operator === 'gt' || filter.operator === 'gte') values[0] = raw
      else if (filter.operator === 'lt' || filter.operator === 'lte') values[1] = raw
      result[filter.field] = values
      continue
    }
    result[filter.field] = filter.operator === 'in' ? (filter.values ?? []).map(valueToken) : filter.value === undefined || filter.value === null ? [] : [valueToken(filter.value)]
  }
  return result
}

watch(() => props.filters, (filters) => { draft.value = draftFromFilters(filters) }, { immediate: true, deep: true })

function updateDraft(field: Field, value: string[], index?: number): void {
  const values = [...(draft.value[field.Key] ?? [])]
  if (index === undefined) draft.value = { ...draft.value, [field.Key]: value }
  else {
    values[index] = value[0] ?? ''
    draft.value = { ...draft.value, [field.Key]: values }
  }
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

function numericRange(field: Field): NumericFilterRange | undefined {
  return props.filterRanges?.[field.Key]
}

function numericValues(field: Field): string[] {
  const values = selectedValues(field)
  return [values[0] ?? '', values[1] ?? '']
}

function numericValueValid(field: Field, raw: string): boolean {
  if (!raw) return true
  const pattern = field.Type === 'integer' ? /^-?\d+$/ : /^-?\d+(?:\.\d+)?$/
  if (!pattern.test(raw)) return false
  const range = numericRange(field)
  if (!range) return false
  if (range && (compareNumeric(raw, range.minimum, field.Type) < 0 || compareNumeric(raw, range.maximum, field.Type) > 0)) return false
  return true
}

function decimalParts(value: string): { scaled: bigint; scale: number } {
  const negative = value.startsWith('-')
  const unsigned = negative ? value.slice(1) : value
  const [whole, fraction = ''] = unsigned.split('.')
  const digits = `${whole}${fraction}`.replace(/^0+(?=\d)/, '')
  const magnitude = BigInt(digits || '0')
  return { scaled: negative ? -magnitude : magnitude, scale: fraction.length }
}

function compareNumeric(left: string, right: string, type: Field['Type']): number {
  if (type === 'integer') {
    const a = BigInt(left)
    const b = BigInt(right)
    return a < b ? -1 : a > b ? 1 : 0
  }
  const a = decimalParts(left)
  const b = decimalParts(right)
  const scale = Math.max(a.scale, b.scale)
  const leftScaled = a.scaled * 10n ** BigInt(scale - a.scale)
  const rightScaled = b.scaled * 10n ** BigInt(scale - b.scale)
  return leftScaled < rightScaled ? -1 : leftScaled > rightScaled ? 1 : 0
}

function numericRangeValid(field: Field): boolean {
  if (field.Type !== 'integer' && field.Type !== 'decimal') return true
  const [minimum, maximum] = numericValues(field)
  if (!numericValueValid(field, minimum) || !numericValueValid(field, maximum)) return false
  return !minimum || !maximum || compareNumeric(minimum, maximum, field.Type) <= 0
}

const invalidNumericRanges = computed(() => fields.value.some((field) => !numericRangeValid(field)))

function isChoice(field: Field): boolean {
  return field.Type === 'enum' || field.Type === 'boolean'
}

function choiceValue(field: Field, raw: string): Value {
  if (field.Type === 'boolean') return raw === 'true'
  return field.Enum?.find((option) => valueToken(option.Value) === raw)?.Value ?? raw
}

function filterFor(field: Field, rawValues: string[]): Filter[] {
  if (field.Type === 'integer' || field.Type === 'decimal') {
    const [minimum = '', maximum = ''] = rawValues
    if (!minimum && !maximum) return []
    const numeric = (value: string): Value => value
    if (minimum && maximum && compareNumeric(minimum, maximum, field.Type) === 0) return [{ field: field.Key, operator: 'eq', value: numeric(minimum) }]
    return [
      ...(minimum ? [{ field: field.Key, operator: 'gte' as const, value: numeric(minimum) }] : []),
      ...(maximum ? [{ field: field.Key, operator: 'lte' as const, value: numeric(maximum) }] : []),
    ]
  }
  if (!rawValues.length || rawValues.every((raw) => raw === '')) return []
  const raw = rawValues[0]
  if (field.Type === 'boolean' || field.Type === 'enum') {
    const optionCount = field.Type === 'boolean' ? 2 : field.Enum?.length ?? 0
    if ((field.Type === 'boolean' || field.Type === 'enum') && optionCount > 0 && rawValues.length >= optionCount) return []
    const values = rawValues.map((item) => choiceValue(field, item))
    return [values.length === 1 ? { field: field.Key, operator: 'eq', value: values[0] } : { field: field.Key, operator: 'in', values }]
  }
  if (field.Type === 'date' || field.Type === 'datetime') return [{ field: field.Key, operator: 'eq', value: raw }]
  return [{ field: field.Key, operator: 'contains', value: raw }]
}

function apply(): void {
  if (invalidNumericRanges.value) return
  emit('apply', fields.value.flatMap((field) => filterFor(field, draft.value[field.Key] ?? [])))
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
        <div v-else-if="field.Type === 'integer' || field.Type === 'decimal'" class="crud-filter-range">
          <label class="crud-filter-range-input">
            <span>{{ messages.filterFrom ?? 'De' }}</span>
            <input class="crud-input" type="number" :step="field.Type === 'integer' ? 1 : 'any'" :min="numericRange(field)?.minimum" :max="numericRange(field)?.maximum" :value="numericValues(field)[0]" :disabled="loading || !numericRange(field)" :aria-invalid="!numericRangeValid(field)" @input="updateDraft(field, [($event.target as HTMLInputElement).value], 0)">
          </label>
          <label class="crud-filter-range-input">
            <span>{{ messages.filterTo ?? 'Até' }}</span>
            <input class="crud-input" type="number" :step="field.Type === 'integer' ? 1 : 'any'" :min="numericRange(field)?.minimum" :max="numericRange(field)?.maximum" :value="numericValues(field)[1]" :disabled="loading || !numericRange(field)" :aria-invalid="!numericRangeValid(field)" @input="updateDraft(field, [($event.target as HTMLInputElement).value], 1)">
          </label>
          <small v-if="numericRange(field)" class="crud-filter-range-hint">{{ (messages.filterRangeHint ?? 'Valores disponíveis: {minimum} a {maximum}.').replace('{minimum}', numericRange(field)!.minimum).replace('{maximum}', numericRange(field)!.maximum) }}</small>
          <small v-else class="crud-filter-range-error" role="status">{{ messages.filterRangeUnavailable ?? 'Limites numéricos indisponíveis; este filtro está desativado.' }}</small>
          <small v-if="numericRange(field) && !numericRangeValid(field)" class="crud-filter-range-error" role="alert">{{ (messages.filterRangeInvalid ?? 'Use valores entre {minimum} e {maximum}.').replace('{minimum}', numericRange(field)!.minimum).replace('{maximum}', numericRange(field)!.maximum) }}</small>
        </div>
        <input v-else class="crud-input" :type="inputType(field)" :value="draft[field.Key]?.[0] ?? ''" :placeholder="messages.filterAll ?? 'Todos'" :disabled="loading" @input="updateDraft(field, ($event.target as HTMLInputElement).value ? [($event.target as HTMLInputElement).value] : [])">
      </label>
    </div>
    <footer class="crud-filter-footer">
      <button class="crud-action" data-clear-crud-action="clear-filters" type="button" :disabled="loading" @click="clear">{{ messages.clearFilters ?? 'Limpar filtros' }}</button>
      <button class="crud-action crud-action-primary" data-clear-crud-action="apply-filters" type="button" :disabled="loading || invalidNumericRanges" @click="apply">{{ messages.applyFilters ?? 'Aplicar filtros' }}</button>
    </footer>
  </section>
</template>
