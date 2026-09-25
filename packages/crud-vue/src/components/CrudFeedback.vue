<script setup lang="ts">
import type { CrudFeedback } from '@clear-platform/crud-client'
import type { CrudMessages, Translate } from '../messages.js'

defineProps<{ feedback?: CrudFeedback; messages: CrudMessages; translate: Translate }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <section v-if="feedback" class="crud-feedback" data-clear-crud-part="feedback" :data-clear-crud-status="feedback.kind" :role="feedback.kind === 'success' ? 'status' : 'alert'">
    <p class="crud-feedback-message">{{ translate(feedback.message) }}</p>
    <button class="crud-icon-action" type="button" :aria-label="messages.close" :title="messages.close" @click="emit('close')">×</button>
  </section>
</template>
