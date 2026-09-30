<script setup lang="ts">
import type { Field, LookupOption, Value } from '@clear-platform-br/crud-client'
import CrudLookupField from './CrudLookupField.vue'
import type { CrudMessages, Translate } from '../messages.js'

const props = withDefaults(defineProps<{
  field: Field
  resource: string
  modelValue?: Value
  dependencies?: Record<string, Value>
  disabled: boolean
  required?: boolean
  mobile?: boolean
  controlName?: string
  ariaLabel?: string
  messages: CrudMessages
  translate: Translate
  lookup: (resource: string, field: string, search: string, dependencies?: Record<string, Value>) => Promise<LookupOption[]>
}>(), { modelValue: null, dependencies: undefined, required: false, mobile: false, controlName: '', ariaLabel: undefined })

const emit = defineEmits<{ update: [value: Value] }>()

function accessibleLabel(): string { return props.ariaLabel || props.translate(props.field.Label) }

// Shared input contract: parent fields and one edited detail row emit the same typed value.
function valueFromEvent(field: Field, event: Event): Value {
  const target = event.target as HTMLInputElement
  if (field.Type === 'boolean') return target.checked
  if (field.Type === 'integer') return target.value === '' ? null : Number.parseInt(target.value, 10)
  if (field.Type === 'datetime') return target.value === '' ? null : new Date(target.value).toISOString()
  if (field.Type === 'decimal') return target.value === '' ? null : target.value
  return target.value === '' ? null : target.value
}

function inputType(field: Field): string {
  return ({ boolean: 'checkbox', integer: 'number', decimal: 'text', date: 'date', datetime: 'datetime-local', email: 'email', phone: 'tel' } as Record<string, string>)[field.Type] ?? 'text'
}

function enumControl(field: Field): 'select' | 'radio' | 'segmented' | 'buttons' {
  if (props.mobile) return 'select'
  if (field.EnumControl === 'select' || field.EnumControl === 'radio' || field.EnumControl === 'segmented' || field.EnumControl === 'buttons') return field.EnumControl
  return (field.Enum?.length ?? 0) <= 4 ? 'segmented' : 'select'
}

function enumIndex(field: Field): string {
  const index = field.Enum?.findIndex((option) => option.Value === props.modelValue) ?? -1
  return index >= 0 ? String(index) : ''
}

function enumValue(field: Field, event: Event): Value {
  const index = Number((event.target as HTMLSelectElement).value)
  return field.Enum?.[index]?.Value ?? null
}

function enumGroupName(field: Field): string {
  return ['crud', props.resource, field.Key, props.controlName].filter(Boolean).join('-')
}

function displayValue(field: Field): string | number {
  if (field.Type !== 'datetime' || typeof props.modelValue !== 'string' || props.modelValue === '') return typeof props.modelValue === 'string' || typeof props.modelValue === 'number' ? props.modelValue : ''
  const date = new Date(props.modelValue)
  return Number.isNaN(date.getTime()) ? props.modelValue : date.toISOString().slice(0, 16)
}
</script>

<template>
  <CrudLookupField v-if="field.Type === 'lookup'" :field="field" :resource="resource" :dependencies="dependencies" :model-value="modelValue" :disabled="disabled" :aria-label="accessibleLabel()" :messages="messages" :translate="translate" :lookup="lookup" @update="emit('update', $event)" />
  <div v-else-if="field.Type === 'enum'" class="crud-enum-control">
    <select v-if="enumControl(field) === 'select'" class="crud-input crud-enum-select" :value="enumIndex(field)" :required="required" :disabled="disabled" :aria-label="accessibleLabel()" @change="emit('update', enumValue(field, $event))">
      <option value="" disabled>{{ messages.selectOption ?? 'Selecione…' }}</option>
      <option v-for="(option, index) in field.Enum ?? []" :key="String(option.Value)" :value="index">{{ translate(option.Label) }}</option>
    </select>
    <div v-else-if="enumControl(field) === 'radio'" class="crud-enum crud-enum--radio" role="radiogroup" :aria-label="accessibleLabel()">
      <label v-for="option in field.Enum ?? []" :key="String(option.Value)" class="crud-radio-option">
        <input class="crud-radio-input" type="radio" :name="enumGroupName(field)" :value="String(option.Value)" :checked="modelValue === option.Value" :required="required" :disabled="disabled" @change="emit('update', option.Value)">
        <span>{{ translate(option.Label) }}</span>
      </label>
    </div>
    <div v-else class="crud-enum" :class="`crud-enum--${enumControl(field)}`" role="radiogroup" :aria-label="accessibleLabel()">
      <button v-for="option in field.Enum ?? []" :key="String(option.Value)" class="crud-enum-option" type="button" role="radio" :aria-checked="modelValue === option.Value" :disabled="disabled" @click="emit('update', option.Value)">{{ translate(option.Label) }}</button>
    </div>
  </div>
  <textarea v-else-if="field.Type === 'text'" class="crud-input" :value="displayValue(field)" :required="required" :disabled="disabled" :aria-label="accessibleLabel()" @input="emit('update', valueFromEvent(field, $event))" />
  <input v-else class="crud-input" :type="inputType(field)" :checked="field.Type === 'boolean' ? Boolean(modelValue) : undefined" :value="field.Type === 'boolean' ? undefined : displayValue(field)" :required="required" :disabled="disabled" :aria-label="accessibleLabel()" @input="emit('update', valueFromEvent(field, $event))">
</template>
