<script setup lang="ts">
import { onUnmounted, ref, shallowRef, watch } from 'vue'
import type { Field, LookupOption, Value } from '@clear-platform/crud-client'
import type { Translate } from '../messages.js'

const props = defineProps<{ field: Field; resource: string; modelValue: Value; disabled: boolean; translate: Translate; lookup: (resource: string, field: string, search: string) => Promise<LookupOption[]> }>()
const emit = defineEmits<{ update: [value: Value] }>()
const options = shallowRef<LookupOption[]>([])
const searchText = ref(String(props.modelValue ?? ''))
let timeout: ReturnType<typeof setTimeout> | undefined

function search(value: string) {
  searchText.value = value
  if (value === '') emit('update', null)
  if (timeout) clearTimeout(timeout)
  timeout = setTimeout(async () => {
    try { options.value = await props.lookup(props.resource, props.field.Key, value) } catch { options.value = [] }
  }, 250)
}
function select(option: LookupOption) { emit('update', option.value); searchText.value = option.label; options.value = [] }
watch(() => props.modelValue, (value) => { searchText.value = String(value ?? '') })
onUnmounted(() => { if (timeout) clearTimeout(timeout) })
</script>

<template>
  <div class="crud-lookup" data-clear-crud-part="lookup-field">
    <input class="crud-input" :value="searchText" type="search" autocomplete="off" :disabled="disabled" :aria-label="translate(field.Label)" @input="search(($event.target as HTMLInputElement).value)">
    <ul v-if="options.length" class="crud-lookup-options" role="listbox">
      <li v-for="option in options" :key="String(option.value)" role="option" class="crud-lookup-option">
        <button class="crud-lookup-option-button" type="button" @click="select(option)">{{ option.label }}</button>
      </li>
    </ul>
  </div>
</template>
