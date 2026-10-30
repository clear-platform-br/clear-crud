<script setup lang="ts">
import { computed } from 'vue'
import type { CrudRecord } from '@clear-platform-br/crud-client'
import ReportDetailsContent from './ReportDetailsContent.vue'

const props = defineProps<{
  recordId: string
  lines: CrudRecord[]
  loading: boolean
  failed: boolean
}>()

const documentName = computed(() => {
  const value = props.lines[0]?.Fields.document_name
  return typeof value === 'string' && value.length > 0 ? value : `registro ${props.recordId}`
})
</script>

<template>
  <section class="report-details-inline" :aria-label="`Detalhes de ${documentName}`">
    <h3>Linhas de {{ documentName }}</h3>
    <ReportDetailsContent :record-id="recordId" :lines="lines" :loading="loading" :failed="failed" />
  </section>
</template>

<style scoped>
.report-details-inline { border-inline-start: 3px solid var(--clear-crud-action-primary, #1f7a68); }
.report-details-inline h3 { margin: 0 0 0.65rem; font-size: 1rem; }
</style>
