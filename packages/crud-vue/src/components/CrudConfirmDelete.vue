<script setup lang="ts">
import { computed, onMounted, useTemplateRef } from 'vue'
import type { CrudRecord } from '@clear-platform/crud-client'
import type { CrudMessages, Translate } from '../messages.js'

const props = defineProps<{ record: CrudRecord; mode: 'archive' | 'hard_delete'; messages: CrudMessages; translate: Translate; busy: boolean }>()
const emit = defineEmits<{ cancel: []; confirm: [] }>()
const cancelButton = useTemplateRef<HTMLButtonElement>('cancelButton')
onMounted(() => cancelButton.value?.focus())
const archive = computed(() => props.mode === 'archive')
const title = computed(() => archive.value ? props.messages.confirmArchive : props.messages.confirmRemove)
const body = computed(() => archive.value ? props.messages.confirmArchiveBody : props.messages.confirmRemoveBody)
const action = computed(() => archive.value ? props.messages.archive : props.messages.remove)
</script>

<template>
  <div class="crud-dialog-backdrop" data-clear-crud-part="confirm-delete">
    <section class="crud-dialog" role="alertdialog" aria-modal="true" :aria-label="title">
      <h2 class="crud-dialog-title">{{ title }}</h2>
      <p class="crud-dialog-body">{{ body }}</p>
      <footer class="crud-dialog-footer">
        <button ref="cancelButton" class="crud-action" type="button" :disabled="busy" @click="emit('cancel')">{{ messages.cancel }}</button>
        <button class="crud-action" :class="{ 'crud-action-destructive': !archive }" type="button" :disabled="busy" @click="emit('confirm')">{{ action }}</button>
      </footer>
    </section>
  </div>
</template>
