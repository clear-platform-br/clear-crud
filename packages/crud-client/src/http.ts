import type { CrudRecord, CrudTransport, Field, LookupOption, Mutation, Page, PublicDefinition, Query, Value } from './types.js'

interface Envelope<T> { data: T; meta?: { page?: number; size?: number; total?: number; nextCursor?: string } }
interface ErrorEnvelope { error?: { code?: string; message?: string; fields?: Record<string, string>; correlationId?: string } }

export class CrudRequestError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly fields?: Record<string, string>,
    readonly correlationId?: string,
  ) {
    super(message)
    this.name = 'CrudRequestError'
  }
}

export interface HttpCrudClientOptions {
  baseUrl?: string
  fetch?: typeof globalThis.fetch
  unavailableMessage?: string
}

export class HttpCrudClient implements CrudTransport {
  private readonly baseUrl: string
  private readonly fetcher: typeof globalThis.fetch
  private readonly unavailableMessage: string

  constructor(options: HttpCrudClientOptions = {}) {
    this.baseUrl = (options.baseUrl ?? '/api/v1/crud').replace(/\/$/, '')
    this.fetcher = options.fetch ?? globalThis.fetch.bind(globalThis)
    this.unavailableMessage = options.unavailableMessage ?? 'crud.ui.unavailable'
  }

  async definition(resource: string, signal?: AbortSignal): Promise<PublicDefinition> {
    return normalizeDefinition(await this.request(`${this.resourceUrl(resource)}/definition`, { signal }))
  }

  async list(resource: string, query: Query, signal?: AbortSignal): Promise<Page> {
    const params = new URLSearchParams()
    if (query.search) params.set('q', query.search)
    if (query.page) params.set('page', String(query.page))
    if (query.size) params.set('size', String(query.size))
    const suffix = params.size ? `?${params}` : ''
    const envelope = await this.requestEnvelope<unknown[]>(`${this.resourceUrl(resource)}/records${suffix}`, { signal })
    return {
      records: envelope.data.map(normalizeRecord),
      page: envelope.meta?.page ?? query.page ?? 1,
      size: envelope.meta?.size ?? query.size ?? 25,
      total: envelope.meta?.total,
      nextCursor: envelope.meta?.nextCursor,
    }
  }

  async get(resource: string, id: string, signal?: AbortSignal): Promise<CrudRecord> {
    return normalizeRecord(await this.request(`${this.resourceUrl(resource)}/records/${encodeURIComponent(id)}`, { signal }))
  }

  async create(resource: string, mutation: Mutation, signal?: AbortSignal): Promise<CrudRecord> {
    return normalizeRecord(await this.request(`${this.resourceUrl(resource)}/records`, { method: 'POST', body: JSON.stringify(mutation), signal }))
  }

  async update(resource: string, id: string, version: number, mutation: Mutation, signal?: AbortSignal): Promise<CrudRecord> {
    return normalizeRecord(await this.request(`${this.resourceUrl(resource)}/records/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify({ version, ...mutation }), signal }))
  }

  async delete(resource: string, id: string, version: number, signal?: AbortSignal): Promise<void> {
    await this.request(`${this.resourceUrl(resource)}/records/${encodeURIComponent(id)}`, { method: 'DELETE', body: JSON.stringify({ version, fields: {} }), signal })
  }

  async lookup(resource: string, field: string, search: string, signal?: AbortSignal): Promise<LookupOption[]> {
    const params = new URLSearchParams()
    if (search) params.set('q', search)
    const suffix = params.size ? `?${params}` : ''
    const value = await this.request<unknown[]>(`${this.resourceUrl(resource)}/lookups/${encodeURIComponent(field)}${suffix}`, { signal })
    return value.map((option) => {
      const raw = asObject(option)
      return { value: (raw.Value ?? raw.value ?? null) as Value, label: String(raw.Label ?? raw.label ?? '') }
    })
  }

  private async request<T>(url: string, init: RequestInit): Promise<T> {
    return (await this.requestEnvelope<T>(url, init)).data
  }

  private async requestEnvelope<T>(url: string, init: RequestInit): Promise<Envelope<T>> {
    let response: Response
    try {
      response = await this.fetcher(url, { ...init, headers: { Accept: 'application/json', ...(init.body ? { 'Content-Type': 'application/json' } : {}), ...init.headers } })
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') throw error
      throw new CrudRequestError('temporarily_unavailable', this.unavailableMessage)
    }
    if (response.status === 204) return { data: undefined as T }
    const payload = await response.json().catch(() => ({})) as Envelope<T> & ErrorEnvelope
    if (!response.ok || !('data' in payload)) {
      const error = payload.error
      throw new CrudRequestError(error?.code ?? 'temporarily_unavailable', error?.message ?? this.unavailableMessage, error?.fields, error?.correlationId)
    }
    return payload
  }

  private resourceUrl(resource: string): string {
    return `${this.baseUrl}/${encodeURIComponent(resource)}`
  }
}

function normalizeDefinition(value: unknown): PublicDefinition {
  const raw = asObject(value)
  const fields = asArray(raw.Fields ?? raw.fields).map(normalizeField)
  return {
    Key: String(raw.Key ?? raw.key ?? ''),
    Labels: normalizeLabels(raw.Labels ?? raw.labels),
    Fields: fields,
    Details: asArray(raw.Details ?? raw.details).map((detail) => {
      const item = asObject(detail)
      return {
        Key: String(item.Key ?? item.key ?? ''), Resource: String(item.Resource ?? item.resource ?? ''), Labels: normalizeLabels(item.Labels ?? item.labels),
        Fields: asArray(item.Fields ?? item.fields).map(normalizeField), Minimum: Number(item.Minimum ?? item.minimum ?? 0), Maximum: Number(item.Maximum ?? item.maximum ?? 0),
        AllowCreate: Boolean(item.AllowCreate ?? item.allowCreate), AllowUpdate: Boolean(item.AllowUpdate ?? item.allowUpdate), AllowDelete: Boolean(item.AllowDelete ?? item.allowDelete),
      }
    }),
    List: normalizeList(raw.List ?? raw.list),
    Presentation: normalizePresentation(raw.Presentation ?? raw.presentation),
    Actions: asArray(raw.Actions ?? raw.actions).map(String) as PublicDefinition['Actions'],
    Delete: normalizeDelete(raw.Delete ?? raw.delete),
  }
}

function normalizeRecord(value: unknown): CrudRecord {
  const raw = asObject(value)
  const details: Record<string, CrudRecord[]> = {}
  for (const [key, records] of Object.entries(asObject(raw.Details ?? raw.details))) details[key] = asArray(records).map(normalizeRecord)
  return { ID: String(raw.ID ?? raw.id ?? ''), Version: Number(raw.Version ?? raw.version ?? 0), Fields: asObject(raw.Fields ?? raw.fields) as Record<string, Value>, Details: Object.keys(details).length ? details : undefined }
}

function normalizeField(value: unknown): Field {
  const raw = asObject(value)
  return {
    Key: String(raw.Key ?? raw.key ?? ''), Label: String(raw.Label ?? raw.label ?? ''), Help: stringOrUndefined(raw.Help ?? raw.help), Type: String(raw.Type ?? raw.type ?? 'string') as Field['Type'],
    Required: Boolean(raw.Required ?? raw.required), ReadOnly: Boolean(raw.ReadOnly ?? raw.readOnly), CreateOnly: Boolean(raw.CreateOnly ?? raw.createOnly), Visible: Boolean(raw.Visible ?? raw.visible), Sensitive: Boolean(raw.Sensitive ?? raw.sensitive),
    MinLength: numberOrUndefined(raw.MinLength ?? raw.minLength), MaxLength: numberOrUndefined(raw.MaxLength ?? raw.maxLength), Minimum: stringOrUndefined(raw.Minimum ?? raw.minimum), Maximum: stringOrUndefined(raw.Maximum ?? raw.maximum),
    Enum: asArray(raw.Enum ?? raw.enum).map((option) => { const item = asObject(option); return { Value: (item.Value ?? item.value ?? null) as Value, Label: String(item.Label ?? item.label ?? '') } }),
    Lookup: raw.Lookup || raw.lookup ? normalizeLookup(raw.Lookup ?? raw.lookup) : undefined,
  }
}

function normalizeLookup(value: unknown): Field['Lookup'] {
  const raw = asObject(value)
  return { Resource: String(raw.Resource ?? raw.resource ?? ''), ValueField: String(raw.ValueField ?? raw.valueField ?? ''), LabelField: String(raw.LabelField ?? raw.labelField ?? ''), Dependencies: asArray(raw.Dependencies ?? raw.dependencies).map(String), PageSize: Number(raw.PageSize ?? raw.pageSize ?? 25) }
}
function normalizeLabels(value: unknown): PublicDefinition['Labels'] { const raw = asObject(value); return { Title: String(raw.Title ?? raw.title ?? ''), Singular: String(raw.Singular ?? raw.singular ?? ''), Help: stringOrUndefined(raw.Help ?? raw.help) } }
function normalizeList(value: unknown): PublicDefinition['List'] { const raw = asObject(value); const pagination = asObject(raw.Pagination ?? raw.pagination); return { Columns: asArray(raw.Columns ?? raw.columns).map(String), Searchable: asArray(raw.Searchable ?? raw.searchable).map(String), Sortable: asArray(raw.Sortable ?? raw.sortable).map(String), DefaultSort: asArray(raw.DefaultSort ?? raw.defaultSort).map((sort) => { const item = asObject(sort); return { Field: String(item.Field ?? item.field ?? ''), Direction: String(item.Direction ?? item.direction ?? 'asc') as 'asc' | 'desc' } }), Pagination: { Mode: String(pagination.Mode ?? pagination.mode ?? 'offset') as 'offset' | 'cursor', DefaultSize: Number(pagination.DefaultSize ?? pagination.defaultSize ?? 25), AllowedSizes: asArray(pagination.AllowedSizes ?? pagination.allowedSizes).map(Number), Total: Boolean(pagination.Total ?? pagination.total) } } }
function normalizePresentation(value: unknown): PublicDefinition['Presentation'] { const raw = asObject(value); return { Collection: String(raw.Collection ?? raw.collection ?? 'auto') as PublicDefinition['Presentation']['Collection'], Density: String(raw.Density ?? raw.density ?? 'comfortable') as PublicDefinition['Presentation']['Density'] } }
function normalizeDelete(value: unknown): PublicDefinition['Delete'] { const raw = asObject(value); const mode = raw.Mode ?? raw.mode; return mode === 'archive' || mode === 'hard_delete' || mode === 'none' ? { Mode: mode } : undefined }
function asObject(value: unknown): Record<string, unknown> { return value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {} }
function asArray(value: unknown): unknown[] { return Array.isArray(value) ? value : [] }
function numberOrUndefined(value: unknown): number | undefined { return typeof value === 'number' ? value : undefined }
function stringOrUndefined(value: unknown): string | undefined { return typeof value === 'string' ? value : undefined }
