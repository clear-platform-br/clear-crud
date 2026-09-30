import { describe, expect, it } from 'vitest'
import { CrudController } from './controller.js'
import { CrudRequestError } from './http.js'
import type { CrudTransport, Mutation, PublicDefinition } from './types.js'

const definition: PublicDefinition = {
  Key: 'contacts', Labels: { Title: 'contacts.title', Singular: 'contacts.singular' },
  Fields: [{ Key: 'name', Label: 'contacts.name', Type: 'string', Required: true, ReadOnly: false, Visible: true, Sensitive: false }],
  Details: [{ Key: 'destinations', Resource: 'contact_destinations', Labels: { Title: 'destinations.title', Singular: 'destinations.singular' }, Fields: [{ Key: 'address', Label: 'destinations.address', Type: 'email', Required: true, ReadOnly: false, Visible: true, Sensitive: false }], Minimum: 1, Maximum: 2, AllowCreate: true, AllowUpdate: true, AllowDelete: true }],
  Grid: { Columns: ['name'], Searchable: ['name'], Sortable: ['name'], DefaultSort: [{ Field: 'name', Direction: 'asc' }], Pagination: { Mode: 'offset', DefaultSize: 25, AllowedSizes: [25], Total: true } }, Form: { Fields: [] },
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
  it('uses the definition default page size for the first grid request', async () => {
    const client = transport()
    let requestedSize: number | undefined
    client.definition = async () => ({ ...definition, Grid: { ...definition.Grid, Pagination: { ...definition.Grid.Pagination, DefaultSize: 10, AllowedSizes: [10] } } })
    client.list = async (_resource, query) => { requestedSize = query.size; return { records: [], page: 1, size: query.size ?? 10, total: 0 } }
    const controller = new CrudController('contacts', client)
    await controller.load()
    expect(requestedSize).toBe(10)
    expect(controller.snapshot().query.size).toBe(10)
  })

  it('reloads the first page when archived visibility changes', async () => {
    const client = transport()
    const queries: Array<{ includeArchived?: boolean; page?: number }> = []
    client.definition = async () => ({ ...definition, Grid: { ...definition.Grid, ArchiveVisibility: 'active_and_archived' } })
    client.list = async (_resource, query) => { queries.push(query); return { records: [], page: query.page ?? 1, size: query.size ?? 25, total: 0 } }
    const controller = new CrudController('contacts', client)
    await controller.load()
    await controller.setArchivedVisibility(true)
    expect(queries.at(-1)).toMatchObject({ includeArchived: true, page: 1 })
    expect(controller.snapshot().query.includeArchived).toBe(true)
  })

  it('reloads the first page with typed column filters', async () => {
    const client = transport()
    const queries: Array<{ filters?: unknown; page?: number }> = []
    client.list = async (_resource, query) => { queries.push(query); return { records: [], page: query.page ?? 1, size: query.size ?? 25, total: 0 } }
    const controller = new CrudController('contacts', client)
    await controller.load()
    await controller.setFilters([{ field: 'name', operator: 'contains', value: 'review' }])
    expect(queries.at(-1)).toMatchObject({ filters: [{ field: 'name', operator: 'contains', value: 'review' }], page: 1 })
    expect(controller.snapshot().query.filters).toEqual([{ field: 'name', operator: 'contains', value: 'review' }])
  })

  it('prefills static defaults for new parent and detail records', async () => {
    const client = transport()
    const defaultDefinition: PublicDefinition = {
      ...definition,
      Fields: [
        { ...definition.Fields[0], Default: 'Novo contato' },
        { Key: 'enabled', Label: 'contacts.enabled', Type: 'boolean', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Default: true },
      ],
      Grid: { ...definition.Grid, Columns: ['name', 'enabled'], Searchable: ['name'], Sortable: ['name'] },
      Details: [{ ...definition.Details[0], Fields: [{ ...definition.Details[0].Fields[0], Default: 'novo@example.com' }] }],
    }
    client.definition = async () => defaultDefinition
    const controller = new CrudController('contacts', client)
    await controller.load()
    controller.beginCreate()
    expect(controller.snapshot().editor?.fields).toMatchObject({ name: 'Novo contato', enabled: true })
    controller.addDetail('destinations')
    expect(controller.snapshot().editor?.details.destinations[0].fields).toEqual({ address: 'novo@example.com' })
  })

  it('resolves fixed child-slot metadata from the parent draft', async () => {
    const client = transport()
    const metadataDefinition: PublicDefinition = {
      ...definition,
      Fields: [
        ...definition.Fields,
        { Key: 'value_label', Label: 'catalog.value_label', Type: 'string', Required: false, ReadOnly: false, Visible: true, Sensitive: false },
        { Key: 'value_type', Label: 'catalog.value_type', Type: 'string', Required: false, ReadOnly: false, Visible: true, Sensitive: false },
        { Key: 'value_required', Label: 'catalog.value_required', Type: 'boolean', Required: false, ReadOnly: false, Visible: true, Sensitive: false },
      ],
      Details: [{
        ...definition.Details[0],
        Fields: [{ ...definition.Details[0].Fields[0], Key: 'value_1', Label: 'crud.value_1', Type: 'string', Required: false }],
        FieldMetadata: [{ Field: 'value_1', LabelField: 'value_label', TypeField: 'value_type', RequiredField: 'value_required' }],
      }],
    }
    client.definition = async () => metadataDefinition
    const controller = new CrudController('contacts', client)
    await controller.load()
    controller.beginCreate()
    controller.updateField('value_label', 'Código')
    controller.updateField('value_type', 'integer')
    controller.updateField('value_required', true)
    controller.addDetail('destinations')
    const editor = controller.snapshot().editor
    expect(editor?.detailFields?.destinations[0]).toMatchObject({ DisplayLabel: 'Código', Type: 'integer', Required: true })
    expect(editor?.details.destinations[0].fields).toEqual({ value_1: null })
  })

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

  it('honors an explicit text maximum before sending the mutation', async () => {
    const client = transport()
    client.definition = async () => ({ ...definition, Fields: [{ ...definition.Fields[0], MaxLength: 50 }] })
    const controller = new CrudController('contacts', client)
    await controller.load()
    controller.beginCreate()
    controller.updateField('name', 'a'.repeat(51))
    await controller.submit()
    expect(client.mutations).toEqual([])
    expect(controller.snapshot().feedback).toMatchObject({ kind: 'error', message: 'crud.ui.validation', fields: { name: 'crud.field.length' } })
  })

	it('normalizes SQLite boolean values before updating a record', async () => {
    const client = transport()
    client.definition = async () => ({ ...definition, Fields: [{ Key: 'enabled', Label: 'contacts.enabled', Type: 'boolean', Required: true, ReadOnly: false, Visible: true, Sensitive: false }, ...definition.Fields], Details: [] })
    client.get = async () => ({ ID: '1', Version: 1, Fields: { enabled: 1, name: 'Ana' } })
    const controller = new CrudController('contacts', client)
    await controller.load()
    await controller.beginEdit({ ID: '1', Version: 1, Fields: { enabled: 1, name: 'Ana' } })
    await controller.submit()
		expect(client.mutations).toEqual([{ fields: { enabled: true, name: 'Ana' }, details: {} }])
	})

	it('keeps a detail-only collection unavailable when the record omits it', async () => {
		const client = transport()
		client.get = async () => ({ ID: '1', Version: 1, Fields: { name: 'Ana' } })
		const controller = new CrudController('contacts', client)
		await controller.load()
		await controller.beginEdit({ ID: '1', Version: 1, Fields: { name: 'Ana' } })
		expect(controller.snapshot().editor?.details).toEqual({})
		await controller.submit()
		expect(client.mutations).toEqual([{ fields: { name: 'Ana' }, details: {} }])
	})

  it('normalizes SQLite boolean values in existing detail rows before updating', async () => {
    const client = transport()
    const detailDefinition: PublicDefinition = {
      ...definition,
      Details: [{
        ...definition.Details[0],
        Fields: [
          { Key: 'active', Label: 'destinations.active', Type: 'boolean', Required: true, ReadOnly: false, Visible: true, Sensitive: false },
          ...definition.Details[0].Fields,
        ],
      }],
    }
    client.definition = async () => detailDefinition
    client.get = async () => ({ ID: '1', Version: 1, Fields: { name: 'Ana' }, Details: { destinations: [{ ID: '2', Version: 1, Fields: { active: 1, address: 'ana@example.com' } }] } })
    const controller = new CrudController('contacts', client)
    await controller.load()
    await controller.beginEdit({ ID: '1', Version: 1, Fields: { name: 'Ana' } })
    await controller.submit()
    expect(client.mutations).toEqual([{ fields: { name: 'Ana' }, details: { destinations: [{ id: '2', version: 1, fields: { active: true, address: 'ana@example.com' } }] } }])
  })

  it('clears every downstream dependent lookup when a parent changes', async () => {
    const client = transport()
    const chainDefinition: PublicDefinition = {
      ...definition,
      Fields: [
        ...definition.Fields,
        { Key: 'state_id', Label: 'state', Type: 'lookup', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Lookup: { Resource: 'states', ValueField: 'id', LabelField: 'name', PageSize: 25 } },
        { Key: 'region_id', Label: 'region', Type: 'lookup', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Lookup: { Resource: 'regions', ValueField: 'id', LabelField: 'name', Dependencies: ['state_id'], PageSize: 25 } },
        { Key: 'city_id', Label: 'city', Type: 'lookup', Required: true, ReadOnly: false, Visible: true, Sensitive: false, Lookup: { Resource: 'cities', ValueField: 'id', LabelField: 'name', Dependencies: ['region_id'], PageSize: 25 } },
      ],
    }
    client.definition = async () => chainDefinition
    const controller = new CrudController('contacts', client)
    await controller.load()
    controller.beginCreate()
    controller.updateField('state_id', 1)
    controller.updateField('region_id', 11)
    controller.updateField('city_id', 111)
    controller.updateField('state_id', 2)
    expect(controller.snapshot().editor?.fields).toMatchObject({ state_id: 2, region_id: null, city_id: null })
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
