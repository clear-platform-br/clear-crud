<script setup lang="ts">
import { computed, onUnmounted, shallowRef, watch } from 'vue'
import type { Field, LookupOption, Value } from '@clear-platform-br/crud-client'
import { ptBR, type CrudMessages, type Translate } from '../messages.js'

const props = withDefaults(defineProps<{ field: Field; resource: string; modelValue: Value; dependencies?: Record<string, Value>; disabled: boolean; messages?: CrudMessages; translate: Translate; lookup: (resource: string, field: string, search: string, dependencies?: Record<string, Value>) => Promise<LookupOption[]> }>(), { messages: () => ptBR })
const emit = defineEmits<{ update: [value: Value] }>()
const options = shallowRef<LookupOption[]>([])
const searchText = shallowRef(String(props.modelValue ?? ''))
const showOptions = shallowRef(false)
const loadingOptions = shallowRef(false)
const lookupAttempted = shallowRef(false)
let timeout: ReturnType<typeof setTimeout> | undefined
let requestNumber = 0
let selectedValue: Value = props.modelValue

const missingDependency = computed(() => (props.field.Lookup?.Dependencies ?? []).some((key) => {
  const value = props.dependencies?.[key]
  return value === null || value === undefined || value === ''
}))
const lookupPageSize = computed(() => props.field.Lookup?.PageSize ?? 25)
const lookupSearchHint = computed(() => (props.messages.lookupSearchHint ?? 'A lista inicial mostra até {size} opções. Digite para buscar outras.').replace('{size}', String(lookupPageSize.value)))
const lookupMinSearchLength = computed(() => {
  const value = props.field.Lookup?.MinSearchLength
  return value && value > 0 ? value : 3
})
const lookupSearchMinimum = computed(() => (props.messages.lookupSearchMinimum ?? 'Digite pelo menos {size} caracteres para pesquisar.').replace('{size}', String(lookupMinSearchLength.value)))
const searchTooShort = computed(() => searchText.value.trim().length > 0 && searchText.value.trim().length < lookupMinSearchLength.value)

async function loadSelectedLabel(value: Value) {
  if (value === null || value === undefined || value === '') return
  const request = ++requestNumber
  try {
    const found = (await props.lookup(props.resource, props.field.Key, String(value), props.dependencies)).find((option) => String(option.value) === String(value))
    if (request === requestNumber && found) searchText.value = found.label
  } catch { /* the user can still type to search */ }
}

function search(value: string) {
  searchText.value = value
  const trimmed = value.trim()
  if (trimmed === '') {
    emit('update', null)
    showOptions.value = false
    options.value = []
    lookupAttempted.value = false
    loadingOptions.value = false
    if (timeout) clearTimeout(timeout)
    return
  }
  if (trimmed.length < lookupMinSearchLength.value) {
    if (timeout) clearTimeout(timeout)
    requestNumber++
    showOptions.value = true
    options.value = []
    lookupAttempted.value = false
    loadingOptions.value = false
    return
  }
  requestOptions(value)
}
function requestOptions(value: string) {
  if (props.disabled || missingDependency.value) return
  if (value.trim() !== '' && value.trim().length < lookupMinSearchLength.value) return
  if (timeout) clearTimeout(timeout)
  showOptions.value = true
  options.value = []
  lookupAttempted.value = true
  loadingOptions.value = true
  timeout = setTimeout(async () => {
    const request = ++requestNumber
    try {
      const result = await props.lookup(props.resource, props.field.Key, value, props.dependencies)
      if (request === requestNumber) options.value = result
    } catch { if (request === requestNumber) options.value = []
    } finally {
      if (request === requestNumber) loadingOptions.value = false
    }
  }, 250)
}
function openAll() {
  if (props.disabled || missingDependency.value) return
  requestOptions('')
}
function handleKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter') return
  event.preventDefault()
  event.stopPropagation()
  if (options.value.length) select(options.value[0])
}
function select(option: LookupOption) { selectedValue = option.value; emit('update', option.value); searchText.value = option.label; showOptions.value = false; options.value = []; lookupAttempted.value = false; loadingOptions.value = false }
watch(() => props.modelValue, (value) => {
  if (String(value ?? '') === String(selectedValue ?? '')) return
  selectedValue = value
  searchText.value = String(value ?? '')
  showOptions.value = false
  options.value = []
  lookupAttempted.value = false
  loadingOptions.value = false
  void loadSelectedLabel(value)
})
watch(() => props.dependencies, () => {
  requestNumber++
  if (timeout) clearTimeout(timeout)
  showOptions.value = false
  options.value = []
  lookupAttempted.value = false
  loadingOptions.value = false
}, { deep: true })
loadSelectedLabel(props.modelValue)
onUnmounted(() => { if (timeout) clearTimeout(timeout) })
</script>

<template>
  <div class="crud-lookup" data-clear-crud-part="lookup-field">
    <div class="crud-lookup-control">
      <input class="crud-input" :value="searchText" type="text" autocomplete="off" :disabled="disabled || missingDependency" :aria-label="translate(field.Label)" role="combobox" :aria-expanded="showOptions" aria-autocomplete="list" @input="search(($event.target as HTMLInputElement).value)" @keydown="handleKeydown">
      <button class="crud-lookup-toggle" type="button" :disabled="disabled || missingDependency" :aria-label="props.messages.lookupOptions ?? 'Abrir opções'" :title="props.messages.lookupOptions ?? 'Abrir opções'" :aria-expanded="showOptions" @click="openAll">⌄</button>
    </div>
    <p v-if="missingDependency" class="crud-lookup-message">{{ props.messages.lookupDependencyRequired ?? 'Escolha uma opção primeiro.' }}</p>
    <p v-else-if="showOptions && searchTooShort" class="crud-lookup-message crud-lookup-search-minimum">{{ lookupSearchMinimum }}</p>
    <p v-else-if="showOptions && lookupAttempted && !loadingOptions && options.length === 0" class="crud-lookup-message">{{ props.messages.lookupNoOptions ?? 'Nenhuma opção disponível.' }}</p>
    <ul v-if="showOptions && options.length" class="crud-lookup-options" role="listbox">
      <li v-for="option in options" :key="String(option.value)" role="option" class="crud-lookup-option">
        <button class="crud-lookup-option-button" type="button" @click="select(option)">{{ option.label }}</button>
      </li>
    </ul>
    <p v-if="showOptions && !loadingOptions && options.length >= lookupPageSize" class="crud-lookup-message crud-lookup-search-hint">{{ lookupSearchHint }}</p>
  </div>
</template>
