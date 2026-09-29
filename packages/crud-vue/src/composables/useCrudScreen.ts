import { CrudController, type CrudRecord, type CrudState, type CrudTransport, type Filter, type Value } from '@clear-platform-br/crud-client'
import { onMounted, onUnmounted, shallowRef } from 'vue'

export function useCrudScreen(resource: string, client: CrudTransport) {
  const controller = new CrudController(resource, client)
  const state = shallowRef<CrudState>(controller.snapshot())
  const unsubscribe = controller.subscribe((next) => { state.value = next })
  onMounted(() => { void controller.load() })
  onUnmounted(unsubscribe)

  return {
    state,
    search: (value: string) => controller.search(value),
    setFilters: (filters: Filter[]) => controller.setFilters(filters),
    toggleArchived: () => controller.setArchivedVisibility(!Boolean(state.value.query.includeArchived)),
    goTo: (page: number) => controller.goTo(page),
    previousPage: () => controller.goTo(Math.max(1, state.value.query.page - 1)),
    nextPage: () => controller.goTo(state.value.query.page + 1),
    beginCreate: () => controller.beginCreate(),
    beginEdit: (record: CrudRecord) => controller.beginEdit(record),
    cancelEdit: () => controller.cancelEdit(),
    updateField: (key: string, value: Value) => controller.updateField(key, value),
    updateDetail: (key: string, index: number, field: string, value: Value) => controller.updateDetail(key, index, field, value),
    addDetail: (key: string) => controller.addDetail(key),
    removeDetail: (key: string, index: number) => controller.removeDetail(key, index),
    submit: () => controller.submit(),
    requestDelete: (record: CrudRecord) => controller.requestDelete(record),
    cancelDelete: () => controller.cancelDelete(),
    confirmDelete: () => controller.confirmDelete(),
    dismissFeedback: () => controller.dismissFeedback(),
    lookup: (lookupResource: string, field: string, search: string, dependencies?: Record<string, Value>) => controller.lookup(lookupResource, field, search, dependencies),
  }
}
