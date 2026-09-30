import { describe, expect, it } from 'vitest'
import { HttpCrudClient } from './http.js'
import { resolveDetailFields } from './detail-metadata.js'

function response(status: number, body: unknown): Response { return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }) }

describe('HttpCrudClient', () => {
  it('normalizes explicit boolean display symbols', async () => {
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: { Key: 'contacts', Labels: {}, Fields: [{ Key: 'enabled', Type: 'boolean', BooleanDisplay: { True: '●', False: '○' } }], Details: [], Grid: {}, Form: {}, Presentation: {}, Actions: [] } }) })
    await expect(client.definition('contacts')).resolves.toMatchObject({ Fields: [{ BooleanDisplay: { True: '●', False: '○' } }] })
  })

  it('normalizes the enum control hint and ignores unknown controls', async () => {
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: { Key: 'contacts', Labels: {}, Fields: [
      { Key: 'status', Type: 'enum', EnumControl: 'select', Enum: [{ Value: 'new', Label: 'Novo' }] },
      { Key: 'other', Type: 'enum', EnumControl: 'chips', Enum: [{ Value: 'new', Label: 'Novo' }] },
      { Key: 'actions', Type: 'enum', EnumControl: 'buttons', Enum: [{ Value: 'new', Label: 'Novo' }] },
    ], Details: [], Grid: {}, Form: {}, Presentation: {}, Actions: [] } }) })
    await expect(client.definition('contacts')).resolves.toMatchObject({ Fields: [{ EnumControl: 'select' }, { EnumControl: undefined }, { EnumControl: 'buttons' }] })
  })

  it('normalizes the lookup search minimum with a safe default', async () => {
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: { Key: 'contacts', Labels: {}, Fields: [
      { Key: 'state_id', Type: 'lookup', Lookup: { Resource: 'states', ValueField: 'id', LabelField: 'name', PageSize: 25 } },
      { Key: 'code', Type: 'lookup', Lookup: { Resource: 'codes', ValueField: 'id', LabelField: 'name', PageSize: 25, MinSearchLength: 2 } },
    ], Details: [], Grid: {}, Form: {}, Presentation: {}, Actions: [] } }) })
    await expect(client.definition('contacts')).resolves.toMatchObject({ Fields: [{ Lookup: { MinSearchLength: 3 } }, { Lookup: { MinSearchLength: 2 } }] })
  })

  it('normalizes the optional contextual title field', async () => {
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: { Key: 'catalogs', Labels: {}, Fields: [], Details: [], Grid: {}, Form: {}, Presentation: { Collection: 'table', Density: 'comfortable', TitleField: 'title' }, Actions: [] } }) })
    await expect(client.definition('catalogs')).resolves.toMatchObject({ Presentation: { TitleField: 'title' } })
  })

  it('preserves detail field metadata from the Go definition and resolves fixed slots', async () => {
    const metadata = [1, 2, 3, 4].map((index) => ({ Field: `value_${index}`, LabelField: `value_${index}_label`, TypeField: `value_${index}_type`, RequiredField: `value_${index}_required` }))
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: {
      Key: 'catalogs', Labels: {}, Fields: [], Details: [{
        Key: 'records', Resource: 'catalog_options', Labels: {},
        Fields: [1, 2, 3, 4].map((index) => ({ Key: `value_${index}`, Label: `crud.value_${index}`, Type: 'string', Required: false, Visible: true, Sensitive: false })),
        FieldMetadata: metadata, Minimum: 0, Maximum: 99, AllowCreate: true, AllowUpdate: true, AllowDelete: true,
      }], Grid: {}, Form: {}, Presentation: {}, Actions: [],
    } }) })
    const definition = await client.definition('catalogs')
    expect(definition.Details[0].FieldMetadata).toEqual(metadata)
    const fields = resolveDetailFields(definition.Details[0], {
      value_1_label: 'Tipo', value_1_type: 'text', value_1_required: true,
      value_2_label: 'Grupo', value_2_type: 'text', value_2_required: false,
      value_3_label: 'Nome', value_3_type: 'text', value_3_required: true,
      value_4_label: '', value_4_type: 'text', value_4_required: false,
    })
    expect(fields.filter((field) => field.Visible).map((field) => field.DisplayLabel)).toEqual(['Tipo', 'Grupo', 'Nome'])
    expect(fields[0]).toMatchObject({ Type: 'text', Required: true })
    expect(fields[3].Visible).toBe(false)
  })

  it('normalizes camelCase detail field metadata', async () => {
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: {
      key: 'catalogs', labels: {}, fields: [], details: [{ key: 'records', resource: 'options', labels: {}, fields: [], fieldMetadata: [{ field: 'value_1', labelField: 'value_1_label', typeField: 'value_1_type', requiredField: 'value_1_required' }] }], grid: {}, form: {}, presentation: {}, actions: [],
    } }) })
    await expect(client.definition('catalogs')).resolves.toMatchObject({ Details: [{ FieldMetadata: [{ Field: 'value_1', LabelField: 'value_1_label', TypeField: 'value_1_type', RequiredField: 'value_1_required' }] }] })
  })

  it('normalizes the Go HTTP envelope and its exported field names', async () => {
    const fetcher: typeof fetch = async () => response(200, {
      data: [{ ID: '1', Version: 2, Fields: { name: 'Ana' }, Details: { destinations: [{ ID: '2', Version: 1, Fields: { address: 'ana@example.com' } }] } }],
      meta: { page: 1, size: 25, total: 1 },
    })
    const client = new HttpCrudClient({ fetch: fetcher })
    await expect(client.list('contacts', {})).resolves.toEqual({ records: [{ ID: '1', Version: 2, Fields: { name: 'Ana' }, Details: { destinations: [{ ID: '2', Version: 1, Fields: { address: 'ana@example.com' } }] } }], page: 1, size: 25, total: 1, nextCursor: undefined })
  })

  it('serializes typed grid filters', async () => {
    let requestedURL = ''
    const client = new HttpCrudClient({ fetch: async (input) => { requestedURL = String(input); return response(200, { data: [] }) } })
    await client.list('contacts', { filters: [
      { field: 'enabled', operator: 'eq', value: false },
      { field: 'status', operator: 'eq', value: 'review' },
      { field: 'status', operator: 'in', values: ['review', 'closed'] },
    ] })
    expect(requestedURL).toContain('filter.enabled.eq=false')
    expect(requestedURL).toContain('filter.status.eq=review')
    expect(requestedURL).toContain('filter.status.in=review&filter.status.in=closed')
  })

  it('serializes the grid search value as q', async () => {
    let requestedURL = ''
    const client = new HttpCrudClient({ fetch: async (input) => { requestedURL = String(input); return response(200, { data: [], meta: { page: 1, size: 10, total: 0 } }) } })
    await client.list('contacts', { search: 'review', page: 1, size: 10 })
    expect(requestedURL).toContain('q=review')
  })

  it('normalizes the public deletion policy without persistence details', async () => {
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: { Key: 'contacts', Labels: {}, Fields: [], Details: [], Grid: {}, Form: {}, Presentation: {}, Actions: ['delete'], Delete: { Mode: 'archive' } } }) })
    await expect(client.definition('contacts')).resolves.toMatchObject({ Delete: { Mode: 'archive' } })
  })

  it('serializes declared lookup dependencies as server-owned query keys', async () => {
    let requestedURL = ''
    const client = new HttpCrudClient({ fetch: async (input) => { requestedURL = String(input); return response(200, { data: [] }) } })
    await client.lookup('contacts', 'region_id', 'São', undefined, { state_id: 1, empty: null })
    expect(requestedURL).toContain('q=S%C3%A3o')
    expect(requestedURL).toContain('depends.state_id=1')
    expect(requestedURL).not.toContain('depends.empty')
  })

  it('serializes the server-declared archived visibility query', async () => {
    let requestedURL = ''
    const client = new HttpCrudClient({ fetch: async (input) => { requestedURL = String(input); return response(200, { data: [] }) } })
    await client.list('contacts', { includeArchived: true })
    expect(requestedURL).toContain('include_archived=true')
  })

  it('treats empty Go field bounds as absent constraints', async () => {
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: {
      Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular', Help: '' },
      Fields: [{ Key: 'name', Label: 'contacts.name', Help: '', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false, MinLength: 0, MaxLength: 0, Minimum: '', Maximum: '' }],
      Details: [], Grid: {}, Form: {}, Presentation: {}, Actions: [], Delete: { Mode: 'none' },
    } }) })
    await expect(client.definition('contacts')).resolves.toMatchObject({
      Labels: { Help: undefined }, Fields: [{ Key: 'name', Help: undefined, MinLength: undefined, MaxLength: undefined, Minimum: undefined, Maximum: undefined }],
    })
  })

  it('sends the explicit detail mutation and returns only a public error', async () => {
    let request: RequestInit | undefined
    const fetcher: typeof fetch = async (_url, init) => { request = init; return response(422, { error: { code: 'validation_failed', message: 'Revise os dados.', fields: { name: 'Obrigatório.' }, correlationId: 'c-1' } }) }
    const client = new HttpCrudClient({ fetch: fetcher })
    await expect(client.create('contacts', { fields: { name: 'Ana' }, details: { destinations: [{ fields: { address: 'ana@example.com' } }] } })).rejects.toMatchObject({ code: 'validation_failed', message: 'Revise os dados.', fields: { name: 'Obrigatório.' }, correlationId: 'c-1' })
    expect(request?.body).toBe('{"fields":{"name":"Ana"},"details":{"destinations":[{"fields":{"address":"ana@example.com"}}]}}')
  })
})
