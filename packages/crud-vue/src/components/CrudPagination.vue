<script setup lang="ts">
import { computed } from 'vue'
import type { Page } from '@clear-platform-br/crud-client'
import type { CrudMessages } from '../messages.js'

const props = defineProps<{ page?: Page; messages: CrudMessages; loading: boolean }>()
const emit = defineEmits<{ previous: []; next: []; goTo: [page: number] }>()

type PageItem = { kind: 'page'; value: number } | { kind: 'ellipsis'; key: string }
function pageItem(value: number): PageItem { return { kind: 'page', value } }
function ellipsis(key: string): PageItem { return { kind: 'ellipsis', key } }

const totalPages = computed(() => {
  if (!props.page || props.page.total === undefined) return undefined
  return Math.max(1, Math.ceil(props.page.total / props.page.size))
})

const pageItems = computed<PageItem[]>(() => {
  const total = totalPages.value
  if (!props.page || total === undefined) return []
  const current = Math.min(Math.max(props.page.page, 1), total)
  if (total <= 7) return Array.from({ length: total }, (_, index) => pageItem(index + 1))
  if (current <= 4) return [1, 2, 3, 4, 5].map(pageItem).concat([ellipsis('end'), pageItem(total)])
  if (current >= total - 3) return [pageItem(1), ellipsis('start')].concat(Array.from({ length: 5 }, (_, index) => pageItem(total - 4 + index)))
  return [pageItem(1), ellipsis('start'), pageItem(current - 1), pageItem(current), pageItem(current + 1), ellipsis('end'), pageItem(total)]
})

const previousDisabled = computed(() => !props.page || props.loading || props.page.page <= 1)
const nextDisabled = computed(() => {
  if (!props.page || props.loading) return true
  if (totalPages.value !== undefined) return props.page.page >= totalPages.value
  return !props.page.nextCursor && props.page.records.length < props.page.size
})

function goTo(page: number): void {
  if (!props.page || props.loading || page === props.page.page) return
  emit('goTo', page)
}
</script>

<template>
  <nav v-if="page" class="crud-pagination" data-clear-crud-part="pagination" :aria-label="messages.search">
    <button class="crud-action crud-pagination-control" type="button" :aria-label="messages.previousPage" :title="messages.previousPage" :disabled="previousDisabled" @click="emit('previous')">‹</button>
    <div v-if="pageItems.length" class="crud-pagination-pages">
      <template v-for="item in pageItems" :key="item.kind === 'page' ? item.value : item.key">
        <button v-if="item.kind === 'page'" class="crud-action crud-pagination-page" :aria-current="item.value === page.page ? 'page' : undefined" type="button" :disabled="loading" @click="goTo(item.value)">{{ item.value }}</button>
        <span v-else class="crud-pagination-ellipsis" aria-hidden="true">…</span>
      </template>
    </div>
    <span v-else class="crud-pagination-status" data-clear-crud-part="pagination-status">{{ page.page }}</span>
    <button class="crud-action crud-pagination-control" type="button" :aria-label="messages.nextPage" :title="messages.nextPage" :disabled="nextDisabled" @click="emit('next')">›</button>
  </nav>
</template>
