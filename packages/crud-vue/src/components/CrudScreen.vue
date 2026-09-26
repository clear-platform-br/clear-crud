<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import type { CrudTransport } from '@clear-platform-br/crud-client'
import CrudCollection from './CrudCollection.vue'
import CrudConfirmDelete from './CrudConfirmDelete.vue'
import CrudEditor from './CrudEditor.vue'
import CrudFeedback from './CrudFeedback.vue'
import CrudHelp from './CrudHelp.vue'
import CrudPagination from './CrudPagination.vue'
import CrudToolbar from './CrudToolbar.vue'
import { createTranslator, ptBR, type CrudMessages } from '../messages.js'
import { useCrudScreen } from '../composables/useCrudScreen.js'

const props = withDefaults(defineProps<{ resource: string; client: CrudTransport; messages?: CrudMessages; resolveMessage?: (code: string) => string | undefined; searchDelay?: number }>(), { messages: () => ptBR, searchDelay: 250 })
const translate = computed(() => createTranslator(props.messages, props.resolveMessage))
const screen = useCrudScreen(props.resource, props.client)
const helpOpen = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | undefined
function search(value: string) { if (searchTimer) clearTimeout(searchTimer); searchTimer = setTimeout(() => { void screen.search(value) }, props.searchDelay) }
onUnmounted(() => { if (searchTimer) clearTimeout(searchTimer) })
const loading = computed(() => screen.state.value.phase === 'loading' || screen.state.value.phase === 'submitting')
</script>

<template>
  <main class="crud-screen" data-clear-crud-part="screen" :data-clear-crud-phase="screen.state.value.phase">
    <CrudToolbar v-if="screen.state.value.definition" :definition="screen.state.value.definition" :messages="props.messages" :translate="translate" :loading="loading" @search="search" @create="screen.beginCreate" @help="helpOpen = true" />
    <p v-else class="crud-loading" role="status">{{ props.messages.loading }}</p>
    <CrudFeedback :feedback="screen.state.value.feedback" :messages="props.messages" :translate="translate" @close="screen.dismissFeedback" />
    <CrudCollection v-if="screen.state.value.definition && screen.state.value.page" :definition="screen.state.value.definition" :records="screen.state.value.page.records" :messages="props.messages" :translate="translate" :loading="loading" @edit="screen.beginEdit" @remove="screen.requestDelete" />
    <CrudPagination :page="screen.state.value.page" :messages="props.messages" :loading="loading" @previous="screen.previousPage" @next="screen.nextPage" />
    <CrudEditor v-if="screen.state.value.definition && screen.state.value.editor" :definition="screen.state.value.definition" :editor="screen.state.value.editor" :feedback="screen.state.value.feedback" :messages="props.messages" :translate="translate" :submitting="loading" :lookup="screen.lookup" @update-field="screen.updateField" @update-detail="screen.updateDetail" @add-detail="screen.addDetail" @remove-detail="screen.removeDetail" @submit="screen.submit" @cancel="screen.cancelEdit" />
    <CrudConfirmDelete v-if="screen.state.value.pendingDelete" :record="screen.state.value.pendingDelete" :mode="screen.state.value.definition?.Delete?.Mode === 'archive' ? 'archive' : 'hard_delete'" :messages="props.messages" :translate="translate" :busy="loading" @cancel="screen.cancelDelete" @confirm="screen.confirmDelete" />
    <CrudHelp v-if="helpOpen && screen.state.value.definition?.Labels.Help" :title="screen.state.value.definition.Labels.Title" :content="screen.state.value.definition.Labels.Help" :messages="props.messages" :translate="translate" @close="helpOpen = false" />
  </main>
</template>
