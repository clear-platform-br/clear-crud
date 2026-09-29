<script setup lang="ts">
import { computed, onUnmounted, ref, shallowRef, watch } from 'vue'
import type { CrudTransport, Filter } from '@clear-platform-br/crud-client'
import CrudCollection from './CrudCollection.vue'
import CrudConfirmDelete from './CrudConfirmDelete.vue'
import CrudEditor from './CrudEditor.vue'
import CrudFeedback from './CrudFeedback.vue'
import CrudFilters from './CrudFilters.vue'
import CrudHelp from './CrudHelp.vue'
import CrudPagination from './CrudPagination.vue'
import CrudToolbar from './CrudToolbar.vue'
import { createTranslator, ptBR, type CrudMessages } from '../messages.js'
import { useCrudScreen } from '../composables/useCrudScreen.js'

const props = withDefaults(defineProps<{ resource: string; client: CrudTransport; messages?: CrudMessages; resolveMessage?: (code: string) => string | undefined; searchDelay?: number }>(), { messages: () => ptBR, searchDelay: 250 })
const translate = computed(() => createTranslator(props.messages, props.resolveMessage))
const screen = useCrudScreen(props.resource, props.client)
const helpOpen = ref(false)
const filtersOpen = shallowRef(false)
type ScreenPreferences = { dark?: boolean; density?: 'comfortable' | 'compact' }
function preferenceKey(resource: string): string { return `clear-crud:${resource}:preferences` }
function readPreferences(resource: string): ScreenPreferences {
  if (typeof window === 'undefined') return {}
  try {
    const value: unknown = JSON.parse(window.localStorage.getItem(preferenceKey(resource)) ?? '{}')
    if (!value || typeof value !== 'object') return {}
    const raw = value as Record<string, unknown>
    return { dark: typeof raw.dark === 'boolean' ? raw.dark : undefined, density: raw.density === 'compact' || raw.density === 'comfortable' ? raw.density : undefined }
  } catch {
    return {}
  }
}
const preferences = readPreferences(props.resource)
const dark = shallowRef(preferences.dark ?? false)
const densityOverride = shallowRef<'comfortable' | 'compact' | undefined>(preferences.density)
const density = computed<'comfortable' | 'compact'>(() => densityOverride.value ?? (screen.state.value.definition?.Presentation.Density === 'compact' ? 'compact' : 'comfortable'))
let searchTimer: ReturnType<typeof setTimeout> | undefined
function search(value: string) { if (searchTimer) clearTimeout(searchTimer); searchTimer = setTimeout(() => { void screen.search(value) }, props.searchDelay) }
function toggleTheme() { dark.value = !dark.value }
function toggleDensity() { densityOverride.value = density.value === 'compact' ? 'comfortable' : 'compact' }
function applyFilters(filters: Filter[]) { filtersOpen.value = false; void screen.setFilters(filters) }
watch([dark, densityOverride], ([nextDark, nextDensity]) => {
  if (typeof window === 'undefined') return
  try { window.localStorage.setItem(preferenceKey(props.resource), JSON.stringify({ dark: nextDark, density: nextDensity })) } catch { /* storage is optional */ }
})
onUnmounted(() => { if (searchTimer) clearTimeout(searchTimer) })
const loading = computed(() => screen.state.value.phase === 'loading' || screen.state.value.phase === 'submitting')
</script>

<template>
  <main class="crud-screen" data-clear-crud-part="screen" :data-clear-crud-phase="screen.state.value.phase" :data-clear-crud-theme="dark ? 'dark' : 'light'" :data-clear-crud-density="density">
    <CrudToolbar v-if="screen.state.value.definition" :definition="screen.state.value.definition" :messages="props.messages" :translate="translate" :loading="loading" :dark="dark" :compact="density === 'compact'" :include-archived="Boolean(screen.state.value.query.includeArchived)" :filter-count="screen.state.value.query.filters?.length ?? 0" :filters-open="filtersOpen" @search="search" @create="screen.beginCreate" @help="helpOpen = true" @toggle-theme="toggleTheme" @toggle-density="toggleDensity" @toggle-archived="screen.toggleArchived" @toggle-filters="filtersOpen = !filtersOpen" />
    <CrudFilters v-if="filtersOpen && screen.state.value.definition" :definition="screen.state.value.definition" :messages="props.messages" :translate="translate" :filters="screen.state.value.query.filters ?? []" :loading="loading" @apply="applyFilters" @close="filtersOpen = false" />
    <p v-if="!screen.state.value.definition" class="crud-loading" role="status">{{ props.messages.loading }}</p>
    <CrudFeedback :feedback="screen.state.value.feedback" :messages="props.messages" :translate="translate" @close="screen.dismissFeedback" />
    <CrudCollection v-if="screen.state.value.definition && screen.state.value.page" :definition="screen.state.value.definition" :records="screen.state.value.page.records" :messages="props.messages" :translate="translate" :loading="loading" :lookup="screen.lookup" @edit="screen.beginEdit" @remove="screen.requestDelete" />
    <CrudPagination :page="screen.state.value.page" :messages="props.messages" :loading="loading" @previous="screen.previousPage" @next="screen.nextPage" @go-to="screen.goTo" />
    <CrudEditor v-if="screen.state.value.definition && screen.state.value.editor" :definition="screen.state.value.definition" :editor="screen.state.value.editor" :feedback="screen.state.value.feedback" :messages="props.messages" :translate="translate" :submitting="loading" :lookup="screen.lookup" @update-field="screen.updateField" @update-detail="screen.updateDetail" @add-detail="screen.addDetail" @remove-detail="screen.removeDetail" @submit="screen.submit" @cancel="screen.cancelEdit" />
    <CrudConfirmDelete v-if="screen.state.value.pendingDelete" :record="screen.state.value.pendingDelete" :mode="screen.state.value.definition?.Delete?.Mode === 'archive' ? 'archive' : 'hard_delete'" :messages="props.messages" :translate="translate" :busy="loading" @cancel="screen.cancelDelete" @confirm="screen.confirmDelete" />
    <CrudHelp v-if="helpOpen && screen.state.value.definition?.Labels.Help" :title="screen.state.value.definition.Labels.Title" :content="screen.state.value.definition.Labels.Help" :messages="props.messages" :translate="translate" @close="helpOpen = false" />
  </main>
</template>
