<script setup lang="ts">
import type { Page } from '@clear-platform-br/crud-client'
import type { CrudMessages } from '../messages.js'

defineProps<{ page?: Page; messages: CrudMessages; loading: boolean }>()
const emit = defineEmits<{ previous: []; next: [] }>()
</script>

<template>
  <nav v-if="page" class="crud-pagination" data-clear-crud-part="pagination" :aria-label="messages.search">
    <button class="crud-action" type="button" :disabled="loading || page.page <= 1" @click="emit('previous')">{{ messages.previousPage }}</button>
    <span class="crud-pagination-status" data-clear-crud-part="pagination-status">{{ page.page }}</span>
    <button class="crud-action" type="button" :disabled="loading || (page.total !== undefined && page.page * page.size >= page.total)" @click="emit('next')">{{ messages.nextPage }}</button>
  </nav>
</template>
