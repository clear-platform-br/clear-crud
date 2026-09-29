<script setup lang="ts">
import type { PublicDefinition } from '@clear-platform-br/crud-client'
import type { CrudMessages, Translate } from '../messages.js'
import { shallowRef } from 'vue'

const props = withDefaults(defineProps<{ definition: PublicDefinition; messages: CrudMessages; translate: Translate; loading: boolean; dark: boolean; compact: boolean; includeArchived?: boolean; filterCount?: number; filtersOpen?: boolean }>(), { includeArchived: false, filterCount: 0, filtersOpen: false })
const emit = defineEmits<{ search: [value: string]; create: []; help: []; toggleTheme: []; toggleDensity: []; toggleArchived: []; toggleFilters: [] }>()
const searchText = shallowRef('')
function updateSearch(value: string): void { searchText.value = value; emit('search', value) }
function submitSearch(): void { emit('search', searchText.value) }
</script>

<template>
  <header class="crud-toolbar" data-clear-crud-part="toolbar">
    <div class="crud-toolbar-heading">
      <h1 class="crud-toolbar-title" data-clear-crud-part="title">{{ translate(definition.Labels.Title) }}</h1>
      <p v-if="definition.Labels.Help" class="crud-toolbar-help">{{ translate(definition.Labels.Help) }}</p>
    </div>
    <div class="crud-toolbar-actions">
      <button v-if="definition.Actions.includes('help') && definition.Labels.Help" class="crud-icon-action" data-clear-crud-action="help" type="button" :aria-label="messages.help" :title="messages.help" @click="emit('help')">?</button>
      <button class="crud-icon-action crud-preference-action" data-clear-crud-action="theme" type="button" :aria-pressed="dark" :aria-label="dark ? (messages.lightMode ?? 'Use light mode') : (messages.darkMode ?? 'Use dark mode')" :title="dark ? (messages.lightMode ?? 'Use light mode') : (messages.darkMode ?? 'Use dark mode')" @click="emit('toggleTheme')"><span aria-hidden="true">{{ dark ? '☀' : '◐' }}</span></button>
      <button class="crud-icon-action crud-preference-action" data-clear-crud-action="density" type="button" :aria-pressed="compact" :aria-label="compact ? (messages.comfortableDensity ?? 'Use comfortable rows') : (messages.compactDensity ?? 'Use compact rows')" :title="compact ? (messages.comfortableDensity ?? 'Use comfortable rows') : (messages.compactDensity ?? 'Use compact rows')" @click="emit('toggleDensity')"><span aria-hidden="true">≡</span></button>
      <button v-if="definition.Grid.Columns.length" class="crud-filter-toggle" data-clear-crud-action="filters" type="button" :aria-expanded="props.filtersOpen" :aria-label="messages.filters ?? 'Filters'" :title="messages.filters ?? 'Filters'" :disabled="loading" @click="emit('toggleFilters')"><span aria-hidden="true">☷</span><span>{{ messages.filters ?? 'Filters' }}</span><span v-if="props.filterCount" class="crud-filter-count">{{ props.filterCount }}</span></button>
      <button v-if="definition.Grid.ArchiveVisibility === 'active_and_archived'" class="crud-archive-toggle" data-clear-crud-action="archive-visibility" role="switch" type="button" :aria-checked="props.includeArchived" :aria-label="messages.includeArchivedHelp ?? 'Include deleted records'" :title="messages.includeArchivedHelp ?? 'Include deleted records'" :disabled="loading" @click="emit('toggleArchived')"><span class="crud-archive-toggle-label">{{ messages.includeArchived ?? 'Excluídos' }}</span><span class="crud-archive-switch" aria-hidden="true"><span class="crud-archive-switch-thumb"></span></span></button>
      <form class="crud-search-form" role="search" @submit.prevent="submitSearch">
        <label class="crud-search-label">
          <span class="crud-visually-hidden">{{ messages.search }}</span>
          <input class="crud-search" data-clear-crud-part="search" type="search" :placeholder="messages.searchPlaceholder" :disabled="loading" @input="updateSearch(($event.target as HTMLInputElement).value)">
        </label>
        <button class="crud-icon-action crud-search-submit" data-clear-crud-action="search-submit" type="submit" :aria-label="messages.search" :title="messages.search" :disabled="loading">
          <svg class="crud-search-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><circle cx="10.8" cy="10.8" r="6.8" /><path d="m16 16 5 5" /></svg>
        </button>
      </form>
      <button v-if="definition.Actions.includes('create')" class="crud-icon-action crud-action-primary crud-create-action" data-clear-crud-action="create" type="button" :aria-label="messages.create" :title="messages.create" :disabled="loading" @click="emit('create')">
        <span aria-hidden="true">+</span>
      </button>
    </div>
  </header>
</template>
