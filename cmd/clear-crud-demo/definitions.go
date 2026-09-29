package main

import (
	"context"
	"database/sql"

	crud "github.com/clear-platform-br/clear-crud"
	"github.com/clear-platform-br/clear-crud/sqladapter"
)

// registerDefinitions is the complete, server-owned catalog of demo CRUDs.
// Add each new resource as one focused register function; do not add routes,
// handlers, or frontend controllers per resource.
// Each example is commented in the same order: resource, rules, relationships,
// projections and registration.
func registerDefinitions(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// 1. Tabela física: o adapter infere o contrato básico do schema.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts"); err != nil {
		return err
	}

	// 1. Tabela física e chave pública do recurso.
	// 2. Paginação e opções traduzidas do campo status.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts",
		sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_ptbr"),
		sqladapter.WithGridPageSize(10),
		sqladapter.WithEnum("status",
			crud.Option{Value: "new", Label: "crud.status.new"},
			crud.Option{Value: "review", Label: "crud.status.review"},
			crud.Option{Value: "closed", Label: "crud.status.closed"},
		),
	); err != nil {
		return err
	}

	// 1. Tabela física e chave pública do recurso.
	// 2. Defaults aplicados pelo servidor na criação.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts",
		sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_defaults"),
		sqladapter.WithDefault("status", "new"),
		sqladapter.WithDefault("enabled", true),
	); err != nil {
		return err
	}

	// 1. Tabela física e chave pública do recurso.
	// 2. Validação declarativa: o nome precisa começar com letra maiúscula.
	// 3. Projeções: mantenha o grid e o formulário nos campos automáticos.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts",
		sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_pattern"),
		sqladapter.WithPattern("name", crud.PatternFirstLetterUpper, "crud.validation.name_uppercase"),
	); err != nil {
		return err
	}

	// 1. Tabela física e chave pública do recurso.
	// 2. Projeções explícitas para grid e formulário.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts",
		sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_ordered"),
		sqladapter.WithGridColumns("status", "name", "enabled", "notes"),
		sqladapter.WithFormFields("status", "name", "enabled", "notes"),
	); err != nil {
		return err
	}

	// 1. Tabela física e tamanho da página.
	// 2. Opções e controles visuais dos campos enum.
	if err := registerEnumControlsDemo(ctx, database, registry); err != nil {
		return err
	}

	// 1. Fontes dos lookups, filtros fixos e catálogos auxiliares.
	if err := registerReferenceCatalogDemo(ctx, database, registry); err != nil {
		return err
	}

	// 1. Modelo de leitura com join, cálculo e campos somente leitura.
	if err := registerReferenceReadModelDemo(ctx, database, registry); err != nil {
		return err
	}

	// 1. Tabela física e chave pública do recurso.
	// 2. Arquivamento, paginação e regras de entrada.
	// 3. Apresentação do booleano e opções do enum.
	// 4. Projeções explícitas do grid e do formulário.
	return sqladapter.RegisterAutoTenantTable(
		ctx,
		database,
		registry,
		"clear_crud_disposable_junk_contacts",
		sqladapter.WithResourceKey("clear_crud_disposable_junk_contacts_enum"),
		sqladapter.WithSoftDelete("archived"),
		sqladapter.WithArchiveVisibility(crud.ArchiveVisibilityActiveAndArchived),
		sqladapter.WithGridPageSize(10),
		sqladapter.WithMaxLength("name", 50),
		sqladapter.WithOptional("notes"),
		sqladapter.WithOptional("zextra_01"),
		sqladapter.WithOptional("zextra_02"),
		sqladapter.WithOptional("zextra_03"),
		sqladapter.WithOptional("zextra_04"),
		sqladapter.WithOptional("zextra_05"),
		sqladapter.WithOptional("zextra_06"),
		sqladapter.WithOptional("zextra_07"),
		sqladapter.WithOptional("zextra_08"),
		sqladapter.WithOptional("zextra_09"),
		sqladapter.WithOptional("zextra_10"),
		sqladapter.WithOptional("zextra_11"),
		sqladapter.WithOptional("zextra_12"),
		sqladapter.WithOptional("zextra_13"),
		sqladapter.WithBooleanDisplay("enabled", "●", "●"),
		sqladapter.WithEnum(
			"status",
			crud.Option{Value: "new", Label: "crud.status.new"},
			crud.Option{Value: "review", Label: "crud.status.review"},
			crud.Option{Value: "closed", Label: "crud.status.closed"},
		),
		sqladapter.WithGridColumns("name", "status", "enabled", "notes"),
		sqladapter.WithFormFields("name", "status", "enabled", "notes"),
	)
}

func registerReferenceCatalogDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// 1. Fontes dos lookups: registre as tabelas como recursos globais.
	for _, table := range []sqladapter.Identifier{"clear_crud_reference_values", "clear_crud_reference_regions", "clear_crud_reference_cities"} {
		if err := sqladapter.RegisterAutoGlobalTable(ctx, database, registry, table, sqladapter.WithGridPageSize(10)); err != nil {
			return err
		}
	}

	// 2. Tabela de Tabelas: registre o pai e seus itens.
	if err := registerAuxiliaryCatalogDemo(ctx, database, registry); err != nil {
		return err
	}

	// 3. Lookups: registre dependências e consumidores com filtros fixos.
	if err := registerReferenceLookupDemo(ctx, database, registry); err != nil {
		return err
	}

	// 4. Consumidores distintos: catálogo selecionável e canais ficam separados.
	if err := registerAuxiliaryCatalogLookupDemo(ctx, database, registry, "clear_crud_auxiliary_catalog_lookup_demo", "10", "20"); err != nil {
		return err
	}
	return registerAuxiliaryChannelLookupDemo(ctx, database, registry)
}

func registerAuxiliaryCatalogLookupDemo(ctx context.Context, database *sql.DB, registry *crud.Registry, resource crud.ResourceKey, catalogIDs ...string) error {
	// 1. Filtro fixo: converta os IDs dos catálogos na operação IN.
	values := make([]crud.Value, len(catalogIDs))
	for index, catalogID := range catalogIDs {
		values[index] = catalogID
	}

	// 2. Lookup: declare a tabela fonte e suas colunas de valor/rótulo.
	return sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_auxiliary_catalog_lookup_demo",
		sqladapter.WithResourceKey(resource),
		sqladapter.WithGridPageSize(10),
		sqladapter.WithLookup("selected_option", crud.LookupDefinition{
			Resource:   "clear_crud_auxiliary_catalog_options",
			ValueField: "value_1",
			LabelField: "value_2",
			FixedFilters: []crud.FixedLookupFilter{
				// Somente itens ativos dos catálogos selecionados são expostos.
				{Field: "catalog_id", Values: values},
				{Field: "active", Values: []crud.Value{true}},
			},
			PageSize: 25,
		}),
		// 3. Projeções: declare as colunas do grid e do formulário.
		sqladapter.WithGridColumns("name", "selected_option"),
		sqladapter.WithFormFields("name", "selected_option"),
	)
}

func registerAuxiliaryChannelLookupDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// 1. Recurso consumidor: reutilize a tabela de itens, fixando o catálogo 30.
	return sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_enum_controls_demo",
		sqladapter.WithResourceKey("clear_crud_auxiliary_catalog_channel_lookup_demo"),
		sqladapter.WithGridPageSize(10),
		sqladapter.WithLookup("channel", crud.LookupDefinition{
			Resource:   "clear_crud_auxiliary_catalog_options",
			ValueField: "value_1",
			LabelField: "value_2",
			FixedFilters: []crud.FixedLookupFilter{
				// 2. Filtro fixo: somente opções ativas de canais.
				{Field: "catalog_id", Values: []crud.Value{"30"}},
				{Field: "active", Values: []crud.Value{true}},
			},
			PageSize: 25,
		}),
		// 3. Projeções: mantenha somente nome e canal.
		sqladapter.WithGridColumns("name", "channel"),
		sqladapter.WithFormFields("name", "channel"),
	)
}

func registerAuxiliaryCatalogDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// 1. Tabela pai: declare metadados, opções e campos apresentados ao usuário.
	catalogs, err := sqladapter.AutoTenantTable(
		ctx,
		database,
		"clear_crud_auxiliary_catalogs",
		// 1.1 Regras, ordenação e opções da tabela pai.
		sqladapter.WithGridPageSize(10),
		sqladapter.WithDefaultSort(
			crud.Sort{Field: "active", Direction: crud.SortAscending},
			crud.Sort{Field: "title", Direction: crud.SortAscending},
		),
		sqladapter.WithEnum("management", crud.Option{Value: "fixed", Label: "crud.catalog.management.fixed"}),
		sqladapter.WithEnum("value_1_type", catalogValueTypeOptions()...),
		sqladapter.WithEnum("value_2_type", catalogValueTypeOptions()...),
		// 1.2 Projeções do grid e do formulário da tabela pai.
		sqladapter.WithGridColumns("title"),
		sqladapter.WithFormFields(
			"title", "code", "management", "max_options", "active",
			"value_1_label", "value_1_type", "value_1_required",
			"value_2_label", "value_2_type", "value_2_required",
		),
	)
	if err != nil {
		return err
	}

	// 2. Tabela filha: declare itens e mantenha catalog_id interno.
	options, err := sqladapter.AutoTenantTable(
		ctx,
		database,
		"clear_crud_auxiliary_catalog_options",
		// 2.1 Regras de paginação, exclusão e ordenação dos itens.
		sqladapter.WithGridPageSize(10),
		sqladapter.WithHardDelete(),
		sqladapter.WithDefaultSort(
			crud.Sort{Field: "catalog_id", Direction: crud.SortAscending},
			crud.Sort{Field: "sort_order", Direction: crud.SortAscending},
		),
	)
	if err != nil {
		return err
	}
	for index := range options.Fields {
		if options.Fields[index].Key == "catalog_id" {
			options.Fields[index].ReadOnly = true
			options.Fields[index].Visible = false
		}
	}

	// 3. Relação: conecte pai e filhos pelo contrato genérico de detalhes.
	catalogs.Details = []crud.DetailDefinition{{Key: "options", Resource: options.Key, ParentField: "catalog_id", Maximum: 99, AllowCreate: true, AllowUpdate: true, AllowDelete: true}}

	// 4. Registro: publique os dois recursos sem rota específica de catálogo.
	if err := registry.Register(ctx, catalogs); err != nil {
		return err
	}
	if err := registry.Register(ctx, options); err != nil {
		return err
	}
	return nil
}

func catalogValueTypeOptions() []crud.Option {
	// Metadados permitidos para os quatro campos livres do catálogo.
	return []crud.Option{{Value: "text", Label: "crud.catalog.type.text"}, {Value: "integer", Label: "crud.catalog.type.integer"}, {Value: "decimal", Label: "crud.catalog.type.decimal"}, {Value: "date", Label: "crud.catalog.type.date"}, {Value: "boolean", Label: "crud.catalog.type.boolean"}, {Value: "lookup", Label: "crud.catalog.type.lookup"}}
}

func registerReferenceLookupDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// 1. Lookup raiz: declare estado e restrinja a kind=state.
	return sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_reference_lookup_demo",
		sqladapter.WithGridPageSize(10),
		sqladapter.WithLookup("state_id", crud.LookupDefinition{
			Resource:     "clear_crud_reference_values",
			ValueField:   "id",
			LabelField:   "name",
			FixedFilters: []crud.FixedLookupFilter{{Field: "kind", Values: []crud.Value{"state"}}},
			PageSize:     25,
		}),
		// 2. Lookups dependentes: o adapter fornece os valores anteriores.
		sqladapter.WithLookup("region_id", crud.LookupDefinition{Resource: "clear_crud_reference_regions", ValueField: "id", LabelField: "name", Dependencies: []crud.FieldKey{"state_id"}, PageSize: 25}),
		sqladapter.WithLookup("city_id", crud.LookupDefinition{Resource: "clear_crud_reference_cities", ValueField: "id", LabelField: "name", Dependencies: []crud.FieldKey{"region_id"}, PageSize: 25}),
		// 3. Projeções: declare as colunas do grid e do formulário.
		sqladapter.WithGridColumns("name", "state_id", "region_id", "city_id"),
		sqladapter.WithFormFields("name", "state_id", "region_id", "city_id"),
	)
}

func registerReferenceReadModelDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// 1. Projeção: mantenha uma lista ordenada para o grid somente leitura.
	columns := []crud.FieldKey{"name", "state_name", "summary", "calculated_number"}

	// 2. Contrato: declare escopo, permissões e campos somente leitura.
	definition := crud.Definition{
		Contract: crud.ContractDefinitionV1,
		Key:      "clear_crud_reference_read_model_demo",
		Labels: crud.Labels{
			Title:    "crud.clear_crud_reference_read_model_demo.title",
			Singular: "crud.clear_crud_reference_read_model_demo.singular",
		},
		Scope:       crud.ScopeRequirements{Mode: crud.ScopeModeTenant, Keys: []string{"tenant_id"}},
		Permissions: crud.Permissions{Read: "crud.clear_crud_reference_read_model_demo.read"},
		Fields: []crud.Field{
			{Key: "name", Label: "crud.field.name", Type: crud.FieldString, Visible: true, ReadOnly: true},
			{Key: "state_name", Label: "crud.field.state_name", Type: crud.FieldString, Visible: true, ReadOnly: true},
			{Key: "summary", Label: "crud.field.summary", Type: crud.FieldString, Visible: true, ReadOnly: true},
			{Key: "calculated_number", Label: "crud.field.calculated_number", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
		},
		Grid: crud.GridDefinition{
			Columns:           columns,
			DefaultSort:       []crud.Sort{{Field: "name", Direction: crud.SortAscending}},
			ArchiveVisibility: crud.ArchiveVisibilityActiveOnly,
			Pagination: crud.PaginationDefinition{
				Mode: crud.PageModeOffset, DefaultSize: 10, AllowedSizes: []uint16{10, 25, 50, 100}, Total: true,
			},
		},
		Presentation: crud.Presentation{Collection: crud.CollectionTable, Density: crud.DensityComfortable},
		Delete:       crud.DeletePolicy{Mode: crud.DeleteModeNone},
		Concurrency:  crud.ConcurrencyPolicy{Mode: crud.ConcurrencyNone},
	}

	// 3. Fonte: registre o SELECT server-owned; o adapter valida aliases,
	// aplica escopo e expõe o resultado sem endpoints de mutação.
	return sqladapter.RegisterReadModel(ctx, database, registry, definition, sqladapter.ReadModelConfig{
		Query: `SELECT c.id AS id, c.tenant_id AS tenant_id, c.name AS name,
            s.name AS state_name, c.name || ' — ' || s.name AS summary,
            (c.state_id * 1000) + c.region_id AS calculated_number
            FROM clear_crud_reference_lookup_demo AS c
            JOIN clear_crud_reference_values AS s
			ON s.id = c.state_id AND s.kind = 'state'`,
		IDColumn: "id",
	})
}

func registerEnumControlsDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// 1. Tabela física e tamanho da página.
	// 2. Opções e controles visuais dos campos enum.
	return sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_enum_controls_demo",
		sqladapter.WithResourceKey("clear_crud_enum_controls_demo"),
		sqladapter.WithGridPageSize(10),
		sqladapter.WithEnum("priority",
			crud.Option{Value: "low", Label: "crud.priority.low"},
			crud.Option{Value: "medium", Label: "crud.priority.medium"},
			crud.Option{Value: "high", Label: "crud.priority.high"},
		),
		sqladapter.WithEnumControl("priority", crud.EnumControlSelect),
		sqladapter.WithEnum("channel",
			crud.Option{Value: "email", Label: "crud.channel.email"},
			crud.Option{Value: "phone", Label: "crud.channel.phone"},
			crud.Option{Value: "whatsapp", Label: "crud.channel.whatsapp"},
		),
		sqladapter.WithEnumControl("channel", crud.EnumControlRadio),
		sqladapter.WithEnum("state",
			crud.Option{Value: "draft", Label: "crud.state.draft"},
			crud.Option{Value: "active", Label: "crud.state.active"},
			crud.Option{Value: "closed", Label: "crud.state.closed"},
		),
		sqladapter.WithEnumControl("state", crud.EnumControlSegmented),
		sqladapter.WithEnum("tone",
			crud.Option{Value: "info", Label: "crud.tone.info"},
			crud.Option{Value: "warning", Label: "crud.tone.warning"},
			crud.Option{Value: "urgent", Label: "crud.tone.urgent"},
		),
		sqladapter.WithEnumControl("tone", crud.EnumControlButtons),
		// 3. Exclusão e projeções: escolha a política e ordene grid/formulário.
		sqladapter.WithHardDelete(),
		sqladapter.WithGridColumns("name", "priority", "channel", "state", "tone"),
		sqladapter.WithFormFields("name", "priority", "channel", "state", "tone"),
	)
}
