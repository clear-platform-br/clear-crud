<script setup lang="ts">
import type { CrudMessages, Translate } from '../messages.js'

defineProps<{ title: string; content: string; messages: CrudMessages; translate: Translate }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <section class="crud-dialog-backdrop" role="presentation">
    <section class="crud-dialog" role="dialog" aria-modal="true" :aria-label="translate(title)">
      <header class="crud-dialog-header"><h2>{{ translate(title) }}</h2><button class="crud-icon-action" type="button" :aria-label="messages.close" :title="messages.close" @click="emit('close')">×</button></header>
      <p>{{ translate(content) }}</p>
      <footer class="crud-dialog-footer"><button class="crud-action crud-action-primary" type="button" @click="emit('close')">{{ messages.close }}</button></footer>
    </section>
  </section>
</template>
