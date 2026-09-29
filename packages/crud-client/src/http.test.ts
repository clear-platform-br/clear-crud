import { describe, expect, it } from 'vitest'
import { HttpCrudClient } from './http.js'

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
