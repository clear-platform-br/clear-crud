import { CrudRequestError } from './http.js'
import { resolveDetailFields } from './detail-metadata.js'
import type { CrudFeedback, CrudRecord, CrudState, CrudTransport, DetailMutation, EditorDraft, Filter, LookupOption, Mutation, PublicDefinition, Value } from './types.js'

export class CrudController {
  private state: CrudState = { phase: 'idle', query: { search: '', filters: [], page: 1, size: 25, includeArchived: false } }
  private listeners = new Set<(state: CrudState) => void>()
  private request?: AbortController

  constructor(readonly resource: string, private readonly transport: CrudTransport) {}

  snapshot(): CrudState { return structuredClone(this.state) }
  subscribe(listener: (state: CrudState) => void): () => void { this.listeners.add(listener); listener(this.snapshot()); return () => this.listeners.delete(listener) }

  async load(): Promise<void> {
    const request = this.beginRequest()
    this.patch({ phase: 'loading', feedback: undefined })
    try {
      const hadDefinition = Boolean(this.state.definition)
      const definition = this.state.definition ?? await this.transport.definition(this.resource, request.signal)
      const query = hadDefinition ? this.state.query : { ...this.state.query, size: definition.Grid.Pagination.DefaultSize }
      if (!hadDefinition) this.patch({ definition, query })
      const page = await this.transport.list(this.resource, query, request.signal)
      if (this.request !== request) return
      this.patch({ definition, page, query, phase: page.records.length ? 'ready' : 'empty' })
    } catch (error) { if (this.request === request) this.handleError(error) } finally { this.completeRequest(request) }
  }

  async search(search: string): Promise<void> { this.patch({ query: { ...this.state.query, search, page: 1 } }); await this.load() }
  async setFilters(filters: Filter[]): Promise<void> { this.patch({ query: { ...this.state.query, filters: filters.map((filter) => ({ ...filter })), page: 1 } }); await this.load() }
  async setArchivedVisibility(includeArchived: boolean): Promise<void> {
    this.patch({ query: { ...this.state.query, includeArchived, page: 1 } })
    await this.load()
  }
  async goTo(page: number): Promise<void> { this.patch({ query: { ...this.state.query, page } }); await this.load() }
  async setPageSize(size: number): Promise<void> { this.patch({ query: { ...this.state.query, size, page: 1 } }); await this.load() }

  beginCreate(): void {
    const definition = this.requireDefinition()
    const fields = blankFields(definition.Fields)
    this.patch({ phase: 'editing', editor: { fields, details: blankDetails(definition), detailFields: effectiveDetailFields(definition, fields) }, feedback: undefined })
  }

  async beginEdit(record: CrudRecord): Promise<void> {
    const request = this.beginRequest()
    this.patch({ phase: 'loading', feedback: undefined })
    try {
      const full = await this.transport.get(this.resource, record.ID, request.signal)
      if (this.request !== request) return
      this.patch({ phase: 'editing', editor: recordToDraft(full, this.requireDefinition()), feedback: undefined })
    } catch (error) { if (this.request === request) this.handleError(error) } finally { this.completeRequest(request) }
  }

  cancelEdit(): void { this.patch({ phase: this.state.page?.records.length ? 'ready' : 'empty', editor: undefined }) }
  updateField(key: string, value: Value): void {
    if (!this.state.editor) return
    const fields = clearLookupDependents(this.requireDefinition().Fields, { ...this.state.editor.fields, [key]: value }, key)
    this.patch({ editor: { ...this.state.editor, fields, detailFields: effectiveDetailFields(this.requireDefinition(), fields) } })
  }
  updateDetail(key: string, index: number, field: string, value: Value): void {
    const editor = this.requireEditor(); const rows = [...(editor.details[key] ?? [])]; const detail = this.requireDefinition().Details.find((item) => item.Key === key); if (!detail || !rows[index]) return
    const fields = clearLookupDependents(editor.detailFields?.[key] ?? detail.Fields, { ...rows[index].fields, [field]: value }, field)
    rows[index] = { ...rows[index], fields }; this.patch({ editor: { ...editor, details: { ...editor.details, [key]: rows } } })
  }
  addDetail(key: string): void { const editor = this.requireEditor(); const detail = this.requireDefinition().Details.find((item) => item.Key === key); const fields = editor.detailFields?.[key] ?? detail?.Fields; if (!detail || !fields || editor.details[key].filter((row) => !row.delete).length >= detail.Maximum) return; this.patch({ editor: { ...editor, details: { ...editor.details, [key]: [...editor.details[key], { fields: blankFields(fields) }] } } }) }
  removeDetail(key: string, index: number): void { const editor = this.requireEditor(); const rows = [...editor.details[key]]; const row = rows[index]; if (!row.id) rows.splice(index, 1); else rows[index] = { ...row, delete: true }; this.patch({ editor: { ...editor, details: { ...editor.details, [key]: rows } } }) }

  async submit(): Promise<void> {
    const editor = this.requireEditor()
    const localErrors = validateDraft(this.requireDefinition(), editor)
    if (Object.keys(localErrors).length) { this.patch({ phase: 'validation_error', feedback: { kind: 'error', message: 'crud.ui.validation', fields: localErrors } }); return }
    const request = this.beginRequest(); this.patch({ phase: 'submitting', feedback: undefined })
    try {
      const mutation: Mutation = { fields: mutationFields(this.requireDefinition(), editor), details: mutationDetails(this.requireDefinition(), editor) }
      const record = editor.id ? await this.transport.update(this.resource, editor.id, editor.version ?? 0, mutation, request.signal) : await this.transport.create(this.resource, mutation, request.signal)
      if (this.request !== request) return
      this.patch({ phase: 'success', editor: undefined, feedback: { kind: 'success', message: 'crud.ui.saved' } })
      await this.load()
      this.patch({ feedback: { kind: 'success', message: 'crud.ui.saved' }, page: replaceRecord(this.state.page, record) })
    } catch (error) { if (this.request === request) this.handleError(error) } finally { this.completeRequest(request) }
  }

  requestDelete(record: CrudRecord): void { this.patch({ pendingDelete: record, feedback: undefined }) }
  cancelDelete(): void { this.patch({ pendingDelete: undefined }) }
  async confirmDelete(): Promise<void> {
    const record = this.state.pendingDelete
    if (!record) return
    const request = this.beginRequest(); this.patch({ phase: 'submitting', feedback: undefined })
    try {
      await this.transport.delete(this.resource, record.ID, record.Version, request.signal)
      if (this.request !== request) return
      this.patch({ pendingDelete: undefined, feedback: { kind: 'success', message: 'crud.ui.removed' } })
      await this.load()
    } catch (error) { if (this.request === request) this.handleError(error) } finally { this.completeRequest(request) }
  }

  async lookup(resource: string, field: string, search: string, dependencies?: Record<string, Value>): Promise<LookupOption[]> { return this.transport.lookup(resource, field, search, undefined, dependencies) }
  dismissFeedback(): void { this.patch({ feedback: undefined }) }

  private patch(change: Partial<CrudState>): void { this.state = { ...this.state, ...change }; for (const listener of this.listeners) listener(this.snapshot()) }
  private beginRequest(): AbortController { this.abortActiveRequest(); const request = new AbortController(); this.request = request; return request }
  private completeRequest(request: AbortController): void { if (this.request === request) this.request = undefined }
  private abortActiveRequest(): void { this.request?.abort(); this.request = undefined }
  private requireDefinition(): PublicDefinition { if (!this.state.definition) throw new Error('CRUD definition is unavailable'); return this.state.definition }
  private requireEditor(): EditorDraft { if (!this.state.editor) throw new Error('CRUD editor is unavailable'); return this.state.editor }
  private handleError(error: unknown): void {
    if (error instanceof DOMException && error.name === 'AbortError') return
    const request = error instanceof CrudRequestError ? error : new CrudRequestError('temporarily_unavailable', 'crud.ui.unavailable')
    const phase = request.code === 'validation_failed' ? 'validation_error' : request.code === 'conflict' ? 'conflict' : 'recoverable_error'
    this.patch({ phase, feedback: { kind: phase === 'conflict' ? 'conflict' : 'error', message: request.message, fields: request.fields } })
  }
}

function blankFields(fields: PublicDefinition['Fields']): Record<string, Value> { return Object.fromEntries(fields.filter((field) => !field.ReadOnly && field.Visible).map((field) => [field.Key, field.Default !== undefined ? field.Default : field.Type === 'boolean' ? false : null])) }
function clearLookupDependents(fields: PublicDefinition['Fields'], values: Record<string, Value>, changedKey: string): Record<string, Value> {
  const next = { ...values }
  const pending = [changedKey]
  while (pending.length) {
    const changed = pending.shift() as string
    for (const field of fields) {
      if (!field.Lookup?.Dependencies?.includes(changed) || next[field.Key] === null || next[field.Key] === undefined) continue
      next[field.Key] = null
      pending.push(field.Key)
    }
  }
  return next
}
function blankDetails(definition: PublicDefinition): Record<string, DetailMutation[]> { return Object.fromEntries(definition.Details.map((detail) => [detail.Key, []])) }
function effectiveDetailFields(definition: PublicDefinition, parentFields: Record<string, Value>): Record<string, PublicDefinition['Details'][number]['Fields']> { return Object.fromEntries(definition.Details.map((detail) => [detail.Key, resolveDetailFields(detail, parentFields)])) }
function recordToDraft(record: CrudRecord, definition: PublicDefinition): EditorDraft {
  const detailFields = effectiveDetailFields(definition, record.Fields)
  return {
    id: record.ID,
    version: record.Version,
    fields: recordFields(definition.Fields, record.Fields),
    details: Object.fromEntries(definition.Details.map((detail) => [
      detail.Key,
      (record.Details?.[detail.Key] ?? []).map((child) => ({
        id: child.ID,
        version: child.Version,
        fields: recordFields(detailFields[detail.Key] ?? detail.Fields, child.Fields),
      })),
    ])),
    detailFields,
  }
}
function recordFields(definition: PublicDefinition['Fields'], values: Record<string, Value>): Record<string, Value> {
  const fields = { ...blankFields(definition), ...values }
  for (const field of definition) fields[field.Key] = draftValue(field, fields[field.Key])
  return fields
}
function draftValue(field: PublicDefinition['Fields'][number], value: Value | undefined): Value {
  if (field.Type === 'boolean' && (value === 0 || value === 1)) return value === 1
  return value ?? null
}
function validateDraft(definition: PublicDefinition, draft: EditorDraft): Record<string, string> {
  const errors: Record<string, string> = {}
  for (const field of definition.Fields) if (!draft.id || !field.CreateOnly) validateField(errors, field.Key, field, draft.fields[field.Key])
  for (const detail of definition.Details) {
    const rows = draft.details[detail.Key] ?? []
    const count = rows.filter((row) => !row.delete).length
    if (count < detail.Minimum || count > detail.Maximum) errors[detail.Key] = 'crud.detail.cardinality'
    for (const row of rows) for (const field of draft.detailFields?.[detail.Key] ?? detail.Fields) if (!row.delete) validateField(errors, `${detail.Key}.${field.Key}`, field, row.fields[field.Key])
  }
  return errors
}
function mutationFields(definition: PublicDefinition, draft: EditorDraft): Record<string, Value> {
  return Object.fromEntries(Object.entries(draft.fields).filter(([key]) => !draft.id || !definition.Fields.find((field) => field.Key === key)?.CreateOnly))
}
function mutationDetails(definition: PublicDefinition, draft: EditorDraft): Record<string, DetailMutation[]> {
  return Object.fromEntries(definition.Details.map((detail) => [detail.Key, (draft.details[detail.Key] ?? []).map((row) => ({
    ...row,
    fields: Object.fromEntries(Object.entries(row.fields).filter(([key]) => {
      const field = draft.detailFields?.[detail.Key]?.find((candidate) => candidate.Key === key) ?? detail.Fields.find((candidate) => candidate.Key === key)
      return field !== undefined && field.Visible && !field.ReadOnly
    })),
  }))]))
}
function validateField(errors: Record<string, string>, key: string, field: PublicDefinition['Fields'][number], value: Value | undefined): void {
  if (field.ReadOnly || empty(value)) { if (field.Required && !field.ReadOnly) errors[key] = 'crud.field.required'; return }
  const text = String(value)
  if (field.Type === 'integer' && (!Number.isInteger(value) || Number.isNaN(value))) errors[key] = 'crud.field.invalid'
  if (field.Type === 'email' && (typeof value !== 'string' || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value))) errors[key] = 'crud.field.invalid'
  if ((field.MinLength !== undefined && text.length < field.MinLength) || (field.MaxLength !== undefined && text.length > field.MaxLength)) errors[key] = 'crud.field.length'
  if (field.Minimum !== undefined || field.Maximum !== undefined) {
    const numeric = Number(value)
    if (Number.isNaN(numeric) || (field.Minimum !== undefined && numeric < Number(field.Minimum)) || (field.Maximum !== undefined && numeric > Number(field.Maximum))) errors[key] = 'crud.field.range'
  }
}
function empty(value: Value | undefined): boolean { return value === null || value === undefined || value === '' }
function replaceRecord(page: CrudState['page'], record: CrudRecord): CrudState['page'] { return page ? { ...page, records: page.records.map((item) => item.ID === record.ID ? record : item) } : page }
