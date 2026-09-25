<script setup lang="ts">
import type { PublicDefinition } from '@clear-platform/crud-client'
import type { CrudMessages, Translate } from '../messages.js'

defineProps<{ definition: PublicDefinition; messages: CrudMessages; translate: Translate; loading: boolean }>()
const emit = defineEmits<{ search: [value: string]; create: []; help: [] }>()
</script>

<template>
  <header class="crud-toolbar" data-clear-crud-part="toolbar">
    <div class="crud-toolbar-heading">
      <h1 class="crud-toolbar-title" data-clear-crud-part="title">{{ translate(definition.Labels.Title) }}</h1>
      <p v-if="definition.Labels.Help" class="crud-toolbar-help">{{ translate(definition.Labels.Help) }}</p>
    </div>
    <div class="crud-toolbar-actions">
      <button v-if="definition.Actions.includes('help') && definition.Labels.Help" class="crud-icon-action" data-clear-crud-action="help" type="button" :aria-label="messages.help" :title="messages.help" @click="emit('help')">?</button>
      <label class="crud-search-label">
        <span class="crud-visually-hidden">{{ messages.search }}</span>
        <input class="crud-search" data-clear-crud-part="search" type="search" :placeholder="messages.searchPlaceholder" :disabled="loading" @input="emit('search', ($event.target as HTMLInputElement).value)">
      </label>
      <button v-if="definition.Actions.includes('create')" class="crud-action crud-action-primary" data-clear-crud-action="create" type="button" :disabled="loading" @click="emit('create')">
        <span aria-hidden="true">+</span><span>{{ messages.create }}</span>
      </button>
    </div>
  </header>
</template>
