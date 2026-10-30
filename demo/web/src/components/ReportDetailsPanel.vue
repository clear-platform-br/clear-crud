<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { CrudRecord } from '@clear-platform-br/crud-client'
import ReportDetailsContent from './ReportDetailsContent.vue'

const props = defineProps<{
  open: boolean
  recordId: string
  lines: CrudRecord[]
  loading: boolean
  failed: boolean
}>()
const emit = defineEmits<{ close: [] }>()
const closeButton = ref<HTMLButtonElement>()
let returnFocusTo: HTMLElement | undefined

const documentName = computed(() => {
  const value = props.lines[0]?.Fields.document_name
  return typeof value === 'string' && value.length > 0 ? value : `registro ${props.recordId}`
})

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close')
}

watch(() => props.open, async (open) => {
  if (open) {
    returnFocusTo = document.activeElement instanceof HTMLElement ? document.activeElement : undefined
    window.addEventListener('keydown', onKeydown)
    await nextTick()
    closeButton.value?.focus()
    return
  }

  window.removeEventListener('keydown', onKeydown)
  await nextTick()
  returnFocusTo?.focus()
  returnFocusTo = undefined
})

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

</script>

<template>
  <div v-if="open" class="crud-dialog-backdrop report-details-backdrop">
    <section class="crud-dialog report-details" role="dialog" aria-modal="true" aria-labelledby="report-details-title">
      <header class="report-details-header">
        <h2 id="report-details-title">Detalhes de {{ documentName }}</h2>
        <button ref="closeButton" class="crud-icon-action" type="button" aria-label="Fechar detalhes" title="Fechar detalhes" @click="emit('close')">×</button>
      </header>
      <ReportDetailsContent :record-id="recordId" :lines="lines" :loading="loading" :failed="failed" />
      <footer class="crud-dialog-footer">
        <button class="crud-action crud-action-primary" type="button" @click="emit('close')">Fechar</button>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.report-details-backdrop { position: absolute; z-index: 50; inset: 0; padding: var(--clear-crud-space-4); }
.report-details { width: min(58rem, 100%); }
.report-details-header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.report-details h2 { margin: 0; font-size: 1.35rem; }
</style>
