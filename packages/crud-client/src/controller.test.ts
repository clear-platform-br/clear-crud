import { describe, expect, it } from 'vitest'
import { CrudController } from './controller.js'
import { CrudRequestError } from './http.js'
import type { CrudTransport, Mutation, PublicDefinition } from './types.js'

const definition: PublicDefinition = {
  Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular' },
  Fields: [{ Key: 'name', Label: 'contacts.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false }],
  Details: [{ Key: 'destinations', Resource: 'contact_destinations', Labels: { Title: 'destinations.title', Singular: 'destinations.singular' }, Fields: [{ Key: 'address', Label: 'destinations.address', Type: 'email', Required: true, ReadOnly: false, Visible: true, Sensitive: false }], Minimum: 1, Maximum: 2, AllowCreate: true, AllowUpdate: true, AllowDelete: true }],
  List: { Columns: ['name'], Searchable: ['name'], Sortable: ['name'], DefaultSort: [{ Field: 'name', Direction: 'asc' }], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } },
  Presentation: { Collection: 'table', Density: 'comfortable' }, Actions: ['create', 'read', 'update', 'delete'],
}

function transport(): CrudTransport & { mutations: Mutation[] } {
  const mutations: Mutation[] = []
  return {
    mutations,
    definition: async () => definition,
    list: async () => ({ records: [], page: 1, size: 25, total: 0 }),
    get: async () => ({ ID: '1', Version: 1, Fields: { name: 'Ana' }, Details: { destinations: [] } }),
    create: async (_resource, mutation) => { mutations.push(mutation); return { ID: '1', Version: 1, Fields: mutation.fields } },
    update: async (_resource, id, version, mutation) => { mutations.push(mutation); return { ID: id, Version: version + 1, Fields: mutation.fields } },
    delete: async () => undefined,
    lookup: async () => [],
  }
}

describe('CrudController', () => {
  it('submits one explicit master-detail mutation', async () => {
    const client = transport()
    const controller = new CrudController('contacts', client)
    await controller.load()
    controller.beginCreate()
    controller.updateField('name', 'Ana')
    controller.addDetail('destinations')
    controller.updateDetail('destinations', 0, 'address', 'ana@example.com')
    await controller.submit()
    expect(client.mutations).toEqual([{ fields: { name: 'Ana' }, details: { destinations: [{ fields: { address: 'ana@example.com' } }] } }])
    expect(controller.snapshot().feedback?.kind).toBe('success')
  })

  it('preserves a validation error without sending a partial mutation', async () => {
    const client = transport()
    const controller = new CrudController('contacts', client)
    await controller.load()
    controller.beginCreate()
    await controller.submit()
    expect(client.mutations).toEqual([])
    expect(controller.snapshot().feedback).toMatchObject({ kind: 'error', message: 'crud.ui.validation', fields: { name: 'crud.field.required' } })
  })

  it('validates typed child values before sending the mutation', async () => {
    const client = transport()
    const controller = new CrudController('contacts', client)
    await controller.load()
    controller.beginCreate()
    controller.updateField('name', 'Ana')
    controller.addDetail('destinations')
    controller.updateDetail('destinations', 0, 'address', 'not-an-email')
    await controller.submit()
    expect(client.mutations).toEqual([])
    expect(controller.snapshot().feedback?.fields).toMatchObject({ 'destinations.address': 'crud.field.invalid' })
  })

  it('submits a create-only identity on create and omits it on update', async () => {
    const client = transport()
    const resourceDefinition: PublicDefinition = { ...definition, Fields: [...definition.Fields, { Key: 'apartment', Label: 'contacts.apartment', Type: 'string', Required: true, ReadOnly: false, CreateOnly: true, Visible: true, Sensitive: false }], Details: definition.Details.map((detail) => ({ ...detail, Minimum: 0 })) }
    client.definition = async () => resourceDefinition
    client.get = async () => ({ ID: '1', Version: 1, Fields: { name: 'Ana', apartment: '64' }, Details: { destinations: [] } })
    const controller = new CrudController('contacts', client)
    await controller.load()
    controller.beginCreate()
    controller.updateField('name', 'Ana')
    controller.updateField('apartment', '64')
    controller.addDetail('destinations')
    controller.updateDetail('destinations', 0, 'address', 'ana@example.com')
    await controller.submit()
    await controller.beginEdit({ ID: '1', Version: 1, Fields: { name: 'Ana' } })
    controller.updateField('name', 'Ana Maria')
    await controller.submit()
    expect(client.mutations).toEqual([
      { fields: { name: 'Ana', apartment: '64' }, details: { destinations: [{ fields: { address: 'ana@example.com' } }] } },
      { fields: { name: 'Ana Maria' }, details: { destinations: [] } },
    ])
  })

  it('maps server failures to a recoverable public state', async () => {
    const client = transport()
    client.list = async () => { throw new CrudRequestError('temporarily_unavailable', 'crud.ui.unavailable') }
    const controller = new CrudController('contacts', client)
    await controller.load()
    expect(controller.snapshot()).toMatchObject({ phase: 'recoverable_error', feedback: { message: 'crud.ui.unavailable' } })
  })
})
