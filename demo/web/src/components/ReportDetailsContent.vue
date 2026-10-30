<script setup lang="ts">
import type { CrudRecord } from '@clear-platform-br/crud-client'

defineProps<{
  recordId: string
  lines: CrudRecord[]
  loading: boolean
  failed: boolean
}>()

function fieldText(record: CrudRecord, field: string): string {
  const value = record.Fields[field]
  return value === null || value === undefined ? '—' : String(value)
}
</script>

<template>
  <div class="report-details-content" aria-live="polite">
    <p class="report-details-help">
      Registro <code>{{ recordId }}</code>. As linhas abaixo foram carregadas pelo host usando o ID recebido; os valores estão em centavos.
    </p>

    <p v-if="loading" class="report-details-state" role="status">Carregando linhas de detalhe…</p>
    <p v-else-if="failed" class="report-details-state report-details-error" role="alert">
      Não foi possível carregar as linhas deste documento.
    </p>
    <p v-else-if="lines.length === 0" class="report-details-state" role="status">
      Nenhuma linha de detalhe encontrada.
    </p>
    <div v-else class="report-details-table-scroll">
      <table aria-label="Linhas do documento">
        <thead>
          <tr>
            <th scope="col">Linha</th>
            <th scope="col">Descrição</th>
            <th scope="col">Valor (centavos)</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="line in lines" :key="line.ID">
            <td>{{ fieldText(line, 'line_number') }}</td>
            <td>{{ fieldText(line, 'description') }}</td>
            <td class="report-details-number">{{ fieldText(line, 'amount_cents') }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.report-details-help { margin: 0 0 0.75rem; color: var(--clear-crud-text-secondary, #52615e); }
.report-details-help code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.report-details-state { margin: 0.75rem 0 0; }
.report-details-error { color: #a02727; }
.report-details-table-scroll { overflow-x: auto; }
.report-details table { width: 100%; border-collapse: collapse; }
.report-details th,
.report-details td { padding: 0.55rem 0.65rem; border-bottom: 1px solid var(--clear-crud-border-default, #cbd7d2); text-align: start; }
.report-details th { font-weight: 700; }
.report-details-number { text-align: end !important; font-variant-numeric: tabular-nums; }
</style>
