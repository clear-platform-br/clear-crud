import { describe, expect, it } from 'vitest'
import { HttpCrudClient } from './http.js'

function response(status: number, body: unknown): Response { return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }) }

describe('HttpCrudClient', () => {
  it('normalizes the Go HTTP envelope and its exported field names', async () => {
    const fetcher: typeof fetch = async () => response(200, {
      data: [{ ID: '1', Version: 2, Fields: { name: 'Ana' }, Details: { destinations: [{ ID: '2', Version: 1, Fields: { address: 'ana@example.com' } }] } }],
      meta: { page: 1, size: 25, total: 1 },
    })
    const client = new HttpCrudClient({ fetch: fetcher })
    await expect(client.list('contacts', {})).resolves.toEqual({ records: [{ ID: '1', Version: 2, Fields: { name: 'Ana' }, Details: { destinations: [{ ID: '2', Version: 1, Fields: { address: 'ana@example.com' } }] } }], page: 1, size: 25, total: 1, nextCursor: undefined })
  })

  it('normalizes the public deletion policy without persistence details', async () => {
    const client = new HttpCrudClient({ fetch: async () => response(200, { data: { Key: 'contacts', Labels: {}, Fields: [], Details: [], List: {}, Presentation: {}, Actions: ['delete'], Delete: { Mode: 'archive' } } }) })
    await expect(client.definition('contacts')).resolves.toMatchObject({ Delete: { Mode: 'archive' } })
  })

  it('sends the explicit detail mutation and returns only a public error', async () => {
    let request: RequestInit | undefined
    const fetcher: typeof fetch = async (_url, init) => { request = init; return response(422, { error: { code: 'validation_failed', message: 'Revise os dados.', fields: { name: 'Obrigatório.' }, correlationId: 'c-1' } }) }
    const client = new HttpCrudClient({ fetch: fetcher })
    await expect(client.create('contacts', { fields: { name: 'Ana' }, details: { destinations: [{ fields: { address: 'ana@example.com' } }] } })).rejects.toMatchObject({ code: 'validation_failed', message: 'Revise os dados.', fields: { name: 'Obrigatório.' }, correlationId: 'c-1' })
    expect(request?.body).toBe('{"fields":{"name":"Ana"},"details":{"destinations":[{"fields":{"address":"ana@example.com"}}]}}')
  })
})
