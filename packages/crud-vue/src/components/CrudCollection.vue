<script setup lang="ts">
import { computed } from 'vue'
import type { CrudRecord, PublicDefinition, Value } from '@clear-platform/crud-client'
import type { CrudMessages, Translate } from '../messages.js'

const props = defineProps<{ definition: PublicDefinition; records: CrudRecord[]; messages: CrudMessages; translate: Translate; loading: boolean }>()
const emit = defineEmits<{ edit: [record: CrudRecord]; remove: [record: CrudRecord] }>()
const columns = computed(() => props.definition.List.Columns.map((key) => props.definition.Fields.find((field) => field.Key === key)).filter((field): field is PublicDefinition['Fields'][number] => Boolean(field)))
function display(value: Value | undefined): string { if (value === null || value === undefined) return '—'; if (typeof value === 'boolean') return value ? '✓' : '—'; return String(value) }
</script>

<template>
  <section class="crud-collection" data-clear-crud-part="collection" :aria-busy="loading">
    <p v-if="!records.length && !loading" class="crud-empty" data-clear-crud-part="empty">{{ messages.noRecords }}</p>
    <div v-else class="crud-table-wrap">
      <table class="crud-table" data-clear-crud-part="table">
        <thead class="crud-table-head"><tr class="crud-table-row"><th v-for="field in columns" :key="field.Key" class="crud-table-heading" scope="col">{{ translate(field.Label) }}</th><th class="crud-table-heading" scope="col"><span class="crud-visually-hidden">{{ messages.edit }}</span></th></tr></thead>
        <tbody class="crud-table-body">
          <tr v-for="record in records" :key="record.ID" class="crud-table-row">
            <td v-for="field in columns" :key="field.Key" class="crud-table-cell" :data-clear-crud-label="translate(field.Label)">{{ display(record.Fields[field.Key]) }}</td>
            <td class="crud-table-cell crud-row-actions">
              <button v-if="definition.Actions.includes('update')" class="crud-icon-action" data-clear-crud-action="edit" type="button" :aria-label="messages.edit" :title="messages.edit" @click="emit('edit', record)">✎</button>
              <button v-if="definition.Actions.includes('delete')" class="crud-icon-action crud-icon-action-destructive" data-clear-crud-action="delete" type="button" :aria-label="messages.remove" :title="messages.remove" @click="emit('remove', record)">×</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
