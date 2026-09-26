export type Action = 'create' | 'read' | 'update' | 'delete' | 'help'
export type FieldType = 'string' | 'text' | 'integer' | 'decimal' | 'boolean' | 'date' | 'datetime' | 'email' | 'phone' | 'enum' | 'lookup'
export type Value = string | number | boolean | null

export interface Field {
  Key: string
  Label: string
  Help?: string
  Type: FieldType
  Required: boolean
  ReadOnly: boolean
  CreateOnly?: boolean
  Visible: boolean
  Sensitive: boolean
  MinLength?: number
  MaxLength?: number
  Minimum?: string
  Maximum?: string
  Enum?: Array<{ Value: Value; Label: string }>
  Lookup?: LookupDefinition
}

export interface LookupDefinition {
  Resource: string
  ValueField: string
  LabelField: string
  Dependencies?: string[]
  PageSize: number
}

export interface DetailDefinition {
  Key: string
  Resource: string
  Labels: { Title: string; Singular: string; Help?: string }
  Fields: Field[]
  Minimum: number
  Maximum: number
  AllowCreate: boolean
  AllowUpdate: boolean
  AllowDelete: boolean
}

export interface PublicDefinition {
  Key: string
  Labels: { Title: string; Singular: string; Help?: string }
  Fields: Field[]
  Details: DetailDefinition[]
  List: {
    Columns: string[]
    Searchable: string[]
    Sortable: string[]
    DefaultSort: Array<{ Field: string; Direction: 'asc' | 'desc' }>
    Pagination: { Mode: 'offset' | 'cursor'; DefaultSize: number; AllowedSizes: number[]; Total: boolean }
  }
  Presentation: { Collection: 'auto' | 'table' | 'cards' | 'list'; Density: 'compact' | 'comfortable' }
  Actions: Action[]
  Delete?: { Mode: 'none' | 'archive' | 'hard_delete' }
}

export interface CrudRecord {
  ID: string
  Version: number
  Fields: Record<string, Value>
  Details?: Record<string, CrudRecord[]>
}

export interface DetailMutation {
  id?: string
  version?: number
  delete?: boolean
  fields: Record<string, Value>
}

export interface Mutation {
  fields: Record<string, Value>
  details?: Record<string, DetailMutation[]>
}

export interface Page {
  records: CrudRecord[]
  page: number
  size: number
  total?: number
  nextCursor?: string
}

export interface Query {
  search?: string
  page?: number
  size?: number
}

export interface LookupOption {
  value: Value
  label: string
}

export interface CrudTransport {
  definition(resource: string, signal?: AbortSignal): Promise<PublicDefinition>
  list(resource: string, query: Query, signal?: AbortSignal): Promise<Page>
  get(resource: string, id: string, signal?: AbortSignal): Promise<CrudRecord>
  create(resource: string, mutation: Mutation, signal?: AbortSignal): Promise<CrudRecord>
  update(resource: string, id: string, version: number, mutation: Mutation, signal?: AbortSignal): Promise<CrudRecord>
  delete(resource: string, id: string, version: number, signal?: AbortSignal): Promise<void>
  lookup(resource: string, field: string, search: string, signal?: AbortSignal): Promise<LookupOption[]>
}

export type CrudPhase = 'idle' | 'loading' | 'ready' | 'empty' | 'editing' | 'submitting' | 'success' | 'validation_error' | 'conflict' | 'recoverable_error'

export interface EditorDraft {
  id?: string
  version?: number
  fields: Record<string, Value>
  details: Record<string, DetailMutation[]>
}

export interface CrudFeedback {
  kind: 'success' | 'error' | 'conflict'
  message: string
  fields?: Record<string, string>
}

export interface CrudState {
  phase: CrudPhase
  definition?: PublicDefinition
  page?: Page
  query: Required<Query>
  editor?: EditorDraft
  pendingDelete?: CrudRecord
  feedback?: CrudFeedback
}
