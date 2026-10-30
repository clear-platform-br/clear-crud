package main

import (
	"context"
	"database/sql"

	crud "github.com/clear-platform-br/clear-crud"
	"github.com/clear-platform-br/clear-crud/sqladapter"
)

const (
	groupedReportResource        = "clear_crud_grouped_report_demo"
	groupedReportDetailsResource = "clear_crud_grouped_report_detail_lines_demo"
)

// registerDefinitions is the complete, server-owned catalog of demo CRUDs.
// The numbered comments identify the corresponding smoke shown by the demo;
// see demo/web/src/definitionSnippets.ts for the human-readable excerpts.
// Each resource is registered here or in one focused helper. Do not add
// resource-specific routes, handlers, or frontend controllers: the same public
// definition, service, adapter and renderer contract serves every smoke.
func registerDefinitions(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// Smoke 01 — mínimo por convenção: o schema fornece metadados para o caso
	// comum; o host ainda aplica o escopo tenant confiável.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts"); err != nil {
		return err
	}

	// Smoke 02 — a mesma tabela física com outra chave lógica, uma página menor
	// e opções localizáveis para status. Os valores persistidos não mudam.
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

	// Smoke extra 01 — defaults escalares aparecem na inclusão e são aplicados
	// pelo core se omitidos; atualização não repõe os defaults.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts",
		sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_defaults"),
		sqladapter.WithDefault("status", "new"),
		sqladapter.WithDefault("enabled", true),
	); err != nil {
		return err
	}

	// Smoke extra 02 — padrão RE2 server-owned para name, com mensagem
	// localizável. O servidor valida criação/edição; não transforma o valor.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts",
		sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_pattern"),
		sqladapter.WithPattern("name", crud.PatternFirstLetterUpper, "crud.validation.name_uppercase"),
	); err != nil {
		return err
	}

	// Smoke extra 03 — a ordem explícita da grade e do formulário é independente
	// da ordem física no schema e da ordenação das linhas.
	if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts",
		sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_ordered"),
		sqladapter.WithGridColumns("status", "name", "enabled", "notes"),
		sqladapter.WithFormFields("status", "name", "enabled", "notes"),
	); err != nil {
		return err
	}

	// Smoke 04 — quatro enums com valores e controles distintos; inclui hard
	// delete explicitamente para demonstrar que exclusão física nunca é default.
	if err := registerEnumControlsDemo(ctx, database, registry); err != nil {
		return err
	}

	// Smokes 05–09 e 11 — registra fontes e consumidores de lookup, dependências,
	// catálogos auxiliares e relações detail-only em recursos fechados no servidor.
	if err := registerReferenceCatalogDemo(ctx, database, registry); err != nil {
		return err
	}

	// Smoke 10 — read model server-owned com join e cálculos somente leitura.
	if err := registerReferenceReadModelDemo(ctx, database, registry); err != nil {
		return err
	}

	// Smoke 12 — declaração de quebras e medidas; o adapter SQL calcula os
	// subtotais sobre todos os registros do escopo/filtros, não só a página.
	if err := registerGroupedReportDemo(ctx, database, registry); err != nil {
		return err
	}

	// Smoke 03 — archive explícito, enum, opções de entrada e projeções. O
	// estado enabled é dado de negócio; archived controla a visibilidade de
	// registros excluídos e não habilita restauração.
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
	// Smoke 05: lookups geográficos usam fontes globais, compartilhadas e
	// somente leitura. O cadastro que referencia essas fontes continua tenant.
	for _, table := range []sqladapter.Identifier{"clear_crud_reference_values", "clear_crud_reference_regions", "clear_crud_reference_cities"} {
		if err := sqladapter.RegisterAutoGlobalTable(ctx, database, registry, table, sqladapter.WithGridPageSize(10)); err != nil {
			return err
		}
	}

	// Smoke 07: pai e filhos ficam na mesma definição mestre-detalhe genérica;
	// os campos de catálogo não criam rotas ou lógica especial no core.
	if err := registerAuxiliaryCatalogDemo(ctx, database, registry); err != nil {
		return err
	}

	// Smoke 05–06: declare lookups dependentes e um filtro fixo de kind.
	if err := registerReferenceLookupDemo(ctx, database, registry); err != nil {
		return err
	}

	// Smoke 08 combina intencionalmente dois catálogos; Smoke 09 restringe a
	// outro consumidor a um catálogo único. Não misture essas intenções.
	if err := registerAuxiliaryCatalogLookupDemo(ctx, database, registry, "clear_crud_auxiliary_catalog_lookup_demo", "10", "20"); err != nil {
		return err
	}
	if err := registerAuxiliaryChannelLookupDemo(ctx, database, registry); err != nil {
		return err
	}

	// Smoke 11: detail-only bloqueia acesso independente ao filho, mantendo
	// metadados/lookup para apresentar o detalhe dentro do pai.
	return registerAuxiliaryDetailLookupDemo(ctx, database, registry)
}

func registerAuxiliaryCatalogLookupDemo(ctx context.Context, database *sql.DB, registry *crud.Registry, resource crud.ResourceKey, catalogIDs ...string) error {
	// Smoke 08. Multi-catalog lookup: transformar os IDs recebidos em Values
	// implementa uma união (IN). Um só ID é igualdade; vários são união, nunca
	// um menu para o operador escolher qual catálogo consultar.
	values := make([]crud.Value, len(catalogIDs))
	for index, catalogID := range catalogIDs {
		values[index] = catalogID
	}

	// O recurso-alvo, as colunas persistida/exibida e os filtros são declarados
	// pelo servidor. A chamada HTTP não pode substituir nenhum deles.
	return sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_auxiliary_catalog_lookup_demo",
		sqladapter.WithResourceKey(resource),
		sqladapter.WithGridPageSize(10),
		sqladapter.WithLookup("selected_option", crud.LookupDefinition{
			Resource:   "clear_crud_auxiliary_catalog_options",
			ValueField: "value_1",
			LabelField: "value_2",
			FixedFilters: []crud.FixedLookupFilter{
				// Somente itens ativos dos catálogos selecionados aparecem como opções.
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
	// Smoke 09. Reutiliza os registros de enum-controls, mas sob outra chave
	// pública e com channel como lookup em vez de enum. catalog_id=30 restringe
	// ao grupo Canais; active=true omite opções inativas.
	return sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_enum_controls_demo",
		sqladapter.WithResourceKey("clear_crud_auxiliary_catalog_channel_lookup_demo"),
		sqladapter.WithGridPageSize(10),
		sqladapter.WithLookup("channel", crud.LookupDefinition{
			Resource:   "clear_crud_auxiliary_catalog_options",
			ValueField: "value_1",
			LabelField: "value_2",
			FixedFilters: []crud.FixedLookupFilter{
				// Valores de canal vêm de value_1; o operador vê value_2.
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

func registerAuxiliaryDetailLookupDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	const (
		parentResource = crud.ResourceKey("clear_crud_auxiliary_catalog_detail_lookup_demo")
		childResource  = crud.ResourceKey("clear_crud_auxiliary_catalog_detail_lookup_options")
	)

	// Smoke 11. Pai: sua grade e formulário só projetam title. O filho será
	// carregado/manipulado como detalhe, e title também contextualiza seu modal.
	catalogs, err := sqladapter.AutoTenantTable(
		ctx,
		database,
		"clear_crud_auxiliary_catalogs",
		sqladapter.WithResourceKey(parentResource),
		sqladapter.WithGridPageSize(10),
		sqladapter.WithDefaultSort(
			crud.Sort{Field: "active", Direction: crud.SortAscending},
			crud.Sort{Field: "title", Direction: crud.SortAscending},
		),
		sqladapter.WithGridColumns("title"),
		sqladapter.WithFormFields("title"),
	)
	if err != nil {
		return err
	}

	// Filho: a tabela física de opções é reutilizada com identidade própria,
	// lookup declarado para value_1 e sem acesso independente (detail-only).
	options, err := sqladapter.AutoTenantTable(
		ctx,
		database,
		"clear_crud_auxiliary_catalog_options",
		sqladapter.WithResourceKey(childResource),
		sqladapter.WithHardDelete(),
		sqladapter.WithLookup("value_1", crud.LookupDefinition{
			Resource:   childResource,
			ValueField: "value_1",
			LabelField: "value_2",
			FixedFilters: []crud.FixedLookupFilter{
				{Field: "catalog_id", Values: []crud.Value{"10"}},
				{Field: "active", Values: []crud.Value{true}},
			},
			PageSize: 25,
		}),
	)
	if err != nil {
		return err
	}
	options.Access = crud.ResourceAccessDetailOnly
	for index := range options.Fields {
		switch options.Fields[index].Key {
		case "catalog_id":
			options.Fields[index].ReadOnly = true
			options.Fields[index].Visible = false
		case "sequence", "active", "sort_order":
			options.Fields[index].Visible = false
		}
	}

	// Relação: ParentAccess testa o valor confiável code do pai, enquanto
	// FixedFilters restringe as opções do lookup. São controles distintos:
	// autorização de abrir/manter a coleção versus seleção permitida no campo.
	// FieldMetadata escolhe labels/requiredness dos slots fixos já existentes.
	catalogs.Details = []crud.DetailDefinition{{
		Key: "options", Resource: childResource, ParentField: "catalog_id", Maximum: 99,
		AllowCreate: true, AllowUpdate: true, AllowDelete: true,
		ParentAccess: &crud.DetailParentAccess{Field: "code", Values: []crud.Value{int64(10)}},
		FieldMetadata: []crud.DetailFieldMetadataSource{
			{Field: "value_1", LabelField: "value_1_label", RequiredField: "value_1_required"},
			{Field: "value_2", LabelField: "value_2_label", RequiredField: "value_2_required"},
		},
	}}
	catalogs.Presentation.TitleField = "title"

	// Registro: ambos continuam usando o caminho genérico registry/service/HTTP;
	// nenhum endpoint ou regra específica de catálogo é introduzido.
	if err := registry.Register(ctx, catalogs); err != nil {
		return err
	}
	return registry.Register(ctx, options)
}

func registerAuxiliaryCatalogDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// Smoke 07. Pai: os metadados descrevem os slots dos itens. O grid apresenta
	// apenas title; o formulário do dev mantém os campos de configuração.
	catalogs, err := sqladapter.AutoTenantTable(
		ctx,
		database,
		"clear_crud_auxiliary_catalogs",
		// 1.1 Comportamento da grade e vocabulários permitidos na configuração.
		sqladapter.WithGridPageSize(10),
		sqladapter.WithDefaultSort(
			crud.Sort{Field: "active", Direction: crud.SortAscending},
			crud.Sort{Field: "title", Direction: crud.SortAscending},
		),
		sqladapter.WithEnum("management", crud.Option{Value: "fixed", Label: "crud.catalog.management.fixed"}),
		sqladapter.WithEnum("value_1_type", catalogValueTypeOptions()...),
		sqladapter.WithEnum("value_2_type", catalogValueTypeOptions()...),
		// 1.2 Separe a visão curta do grid do formulário completo do catálogo.
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

	// Filho: opções guardam os slots de valor; catalog_id é vínculo técnico do
	// adapter, fica oculto e read-only para o operador.
	options, err := sqladapter.AutoTenantTable(
		ctx,
		database,
		"clear_crud_auxiliary_catalog_options",
		// 2.1 Limite de página, política de exclusão e ordem pai/ordem de item.
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

	// Relação: conecte pai e filho com ParentField; mutações do detalhe são parte
	// da mutation do pai e ficam limitadas ao pai que o usuário abriu.
	// FieldMetadata altera apresentação/validação dos slots declarados, não a
	// estrutura: o filho continua tendo slots fixos mesmo se alguns forem ocultos.
	catalogs.Details = []crud.DetailDefinition{{
		Key: "options", Resource: options.Key, ParentField: "catalog_id", Maximum: 99,
		AllowCreate: true, AllowUpdate: true, AllowDelete: true,
		FieldMetadata: []crud.DetailFieldMetadataSource{
			{Field: "value_1", LabelField: "value_1_label", TypeField: "value_1_type", RequiredField: "value_1_required"},
			{Field: "value_2", LabelField: "value_2_label", TypeField: "value_2_type", RequiredField: "value_2_required"},
		},
	}}
	// Contexto: title do registro pai nomeia o modal de seus itens.
	catalogs.Presentation.TitleField = "title"

	// Registro: publique pai e filho no registry comum, sem endpoint ou motor TdT.
	if err := registry.Register(ctx, catalogs); err != nil {
		return err
	}
	if err := registry.Register(ctx, options); err != nil {
		return err
	}
	return nil
}

func catalogValueTypeOptions() []crud.Option {
	// Valores de metadados aceitos pelos slots configuráveis; o catálogo guarda
	// tipos declarativos e o renderer resolve somente os tipos suportados.
	return []crud.Option{{Value: "text", Label: "crud.catalog.type.text"}, {Value: "integer", Label: "crud.catalog.type.integer"}, {Value: "decimal", Label: "crud.catalog.type.decimal"}, {Value: "date", Label: "crud.catalog.type.date"}, {Value: "boolean", Label: "crud.catalog.type.boolean"}, {Value: "lookup", Label: "crud.catalog.type.lookup"}}
}

func registerReferenceLookupDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// Smoke 05 (cadeia) e Smoke 06 (filtro fixo). O browser controla o valor
	// escolhido pelo usuário, nunca o conjunto-base kind=state do primeiro lookup.
	return sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_reference_lookup_demo",
		sqladapter.WithGridPageSize(10),
		sqladapter.WithLookup("state_id", crud.LookupDefinition{
			Resource:     "clear_crud_reference_values",
			ValueField:   "id",
			LabelField:   "name",
			FixedFilters: []crud.FixedLookupFilter{{Field: "kind", Values: []crud.Value{"state"}}},
			PageSize:     25,
		}),
		// Região depende do estado e município depende da região; o serviço só
		// consulta cada filho quando o ancestral necessário está presente.
		sqladapter.WithLookup("region_id", crud.LookupDefinition{Resource: "clear_crud_reference_regions", ValueField: "id", LabelField: "name", Dependencies: []crud.FieldKey{"state_id"}, PageSize: 25}),
		sqladapter.WithLookup("city_id", crud.LookupDefinition{Resource: "clear_crud_reference_cities", ValueField: "id", LabelField: "name", Dependencies: []crud.FieldKey{"region_id"}, PageSize: 25}),
		// IDs permanecem persistidos, mas o renderer resolve labels na grade e
		// usa os mesmos recursos de referência no formulário.
		sqladapter.WithGridColumns("name", "state_id", "region_id", "city_id"),
		sqladapter.WithFormFields("name", "state_id", "region_id", "city_id"),
	)
}

func registerReferenceReadModelDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// Smoke 10. A consulta produz campos de origem, join e cálculo; esta lista
	// decide sua ordem visual sem determinar a ordem das linhas.
	columns := []crud.FieldKey{"name", "state_name", "summary", "calculated_number"}

	// Contrato de leitura: as chaves/aliases têm tipos, escopo e permissão
	// server-owned. ReadOnly impede que o client use os valores de join/cálculo
	// para criar ou atualizar a tabela original.
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

	// Fonte: registre SELECT fixo no bootstrap. O adapter valida aliases e
	// aplica o escopo tenant antes de listar; o navegador não envia SQL.
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

func registerGroupedReportDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// Smoke 12. Este recurso é um relatório imutável: campos, escopo, pesquisa,
	// ordenação, paginação, níveis e medidas compõem uma allowlist server-owned.
	// Os campos continuam declarados e tipados como em qualquer CRUD.
	// NumericDisplay é opcional e só formata a apresentação. O valor armazenado
	// e agregado continua em centavos inteiros, sem símbolo concatenado.
	amountDisplay := crud.NumericDisplay{Locale: "pt-BR", Currency: "BRL", Scale: 2}
	definition := crud.Definition{
		Contract:    crud.ContractDefinitionV1,
		Key:         "clear_crud_grouped_report_demo",
		Labels:      crud.Labels{Title: "crud.clear_crud_grouped_report_demo.title", Singular: "crud.clear_crud_grouped_report_demo.singular"},
		Scope:       crud.ScopeRequirements{Mode: crud.ScopeModeTenant, Keys: []string{"tenant_id"}},
		Permissions: crud.Permissions{Read: "crud.clear_crud_grouped_report_demo.read"},
		Fields: []crud.Field{
			{Key: "year", Label: "crud.field.year", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
			{Key: "month", Label: "crud.field.month", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
			{Key: "document_name", Label: "crud.field.document_name", Type: crud.FieldString, Visible: true, ReadOnly: true},
			{Key: "amount_cents", Label: "crud.field.amount_cents", Type: crud.FieldInteger, Visible: true, ReadOnly: true, NumericDisplay: &amountDisplay},
		},
		Grid: crud.GridDefinition{
			Columns:     []crud.FieldKey{"year", "month", "document_name", "amount_cents"},
			Searchable:  []crud.FieldKey{"document_name"},
			Sortable:    []crud.FieldKey{"year", "month", "document_name", "amount_cents"},
			DefaultSort: []crud.Sort{{Field: "document_name", Direction: crud.SortAscending}},
			Pagination:  crud.PaginationDefinition{Mode: crud.PageModeOffset, DefaultSize: 10, AllowedSizes: []uint16{2, 10, 25}, Total: true},
			Report: &crud.ReportDefinition{
				Groups: []crud.ReportGroup{{Field: "year", Direction: crud.SortAscending}, {Field: "month", Direction: crud.SortAscending}},
				Measures: []crud.ReportMeasure{
					{Key: "documents", Label: "crud.report.documents", Operation: crud.ReportCount},
					{Key: "amount_cents", Label: "crud.report.amount_cents", Operation: crud.ReportSum, Field: "amount_cents", NumericDisplay: &amountDisplay},
				},
			},
		},
		Presentation: crud.Presentation{Collection: crud.CollectionTable, Density: crud.DensityComfortable},
		Delete:       crud.DeletePolicy{Mode: crud.DeleteModeNone},
		Concurrency:  crud.ConcurrencyPolicy{Mode: crud.ConcurrencyNone},
	}

	// Fonte SQL server-owned com linhas sintéticas no próprio SELECT. Assim o
	// smoke não depende de migrations nem escreve no banco SQLite do demo. Os
	// registros beta provam isolamento: não entram nas páginas nem nos totais do
	// tenant alpha. Datas são decompostas em ano/mês pela fonte; o adapter agrupa.
	if err := sqladapter.RegisterReadModel(ctx, database, registry, definition, sqladapter.ReadModelConfig{
		Query: `WITH report_rows(id, tenant_id, document_date, document_name, amount_cents) AS (
			SELECT 1, 'demo_tenant_alpha', '2025-11-01', 'Documento A', 100 UNION ALL
			SELECT 2, 'demo_tenant_alpha', '2025-11-05', 'Documento B', 250 UNION ALL
			SELECT 3, 'demo_tenant_alpha', '2025-11-20', 'Documento C', 300 UNION ALL
			SELECT 4, 'demo_tenant_alpha', '2025-12-05', 'Documento D', 400 UNION ALL
			SELECT 5, 'demo_tenant_alpha', '2026-01-03', 'Documento E', 500 UNION ALL
			SELECT 6, 'demo_tenant_alpha', '2026-01-19', 'Documento F', 600 UNION ALL
			SELECT 7, 'demo_tenant_beta', '2025-11-10', 'Documento de outro tenant', 90000
		)
		SELECT id, tenant_id, CAST(substr(document_date, 1, 4) AS INTEGER) AS year,
			CAST(substr(document_date, 6, 2) AS INTEGER) AS month,
			document_name, amount_cents FROM report_rows`,
		IDColumn:   "id",
		Searchable: []crud.FieldKey{"document_name"},
	}); err != nil {
		return err
	}

	// O mesmo smoke registra linhas relacionadas em outro read model tenant-scoped.
	// O host recebe o ID do documento no evento record-details e busca somente
	// as linhas com esse report_id; o renderer não inicia edição nem mutação.
	detailDefinition := crud.Definition{
		Contract:    crud.ContractDefinitionV1,
		Key:         groupedReportDetailsResource,
		Labels:      crud.Labels{Title: "crud.clear_crud_grouped_report_details.title", Singular: "crud.clear_crud_grouped_report_details.singular"},
		Scope:       crud.ScopeRequirements{Mode: crud.ScopeModeTenant, Keys: []string{"tenant_id"}},
		Permissions: crud.Permissions{Read: "crud.clear_crud_grouped_report_details.read"},
		Fields: []crud.Field{
			// Publicamente filtrável pelo host; não é coluna porque Grid.Columns o omite.
			{Key: "report_id", Label: "crud.field.report_id", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
			{Key: "document_name", Label: "crud.field.document_name", Type: crud.FieldString, Visible: true, ReadOnly: true},
			{Key: "line_number", Label: "crud.field.line_number", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
			{Key: "description", Label: "crud.field.description", Type: crud.FieldString, Visible: true, ReadOnly: true},
			{Key: "amount_cents", Label: "crud.field.amount_cents", Type: crud.FieldInteger, Visible: true, ReadOnly: true, NumericDisplay: &amountDisplay},
		},
		Grid: crud.GridDefinition{
			Columns:     []crud.FieldKey{"line_number", "description", "amount_cents"},
			Searchable:  []crud.FieldKey{"description"},
			Sortable:    []crud.FieldKey{"line_number", "description"},
			DefaultSort: []crud.Sort{{Field: "line_number", Direction: crud.SortAscending}},
			Pagination:  crud.PaginationDefinition{Mode: crud.PageModeOffset, DefaultSize: 25, AllowedSizes: []uint16{25}, Total: true},
		},
		Presentation: crud.Presentation{Collection: crud.CollectionTable, Density: crud.DensityComfortable},
		Delete:       crud.DeletePolicy{Mode: crud.DeleteModeNone},
		Concurrency:  crud.ConcurrencyPolicy{Mode: crud.ConcurrencyNone},
	}

	return sqladapter.RegisterReadModel(ctx, database, registry, detailDefinition, sqladapter.ReadModelConfig{
		Query: `WITH detail_rows(id, tenant_id, report_id, document_name, line_number, description, amount_cents) AS (
			SELECT 101, 'demo_tenant_alpha', 1, 'Documento A', 1, 'Parcela principal', 75 UNION ALL
			SELECT 102, 'demo_tenant_alpha', 1, 'Documento A', 2, 'Complemento', 25 UNION ALL
			SELECT 201, 'demo_tenant_alpha', 2, 'Documento B', 1, 'Parcela principal', 200 UNION ALL
			SELECT 202, 'demo_tenant_alpha', 2, 'Documento B', 2, 'Complemento', 50 UNION ALL
			SELECT 301, 'demo_tenant_alpha', 3, 'Documento C', 1, 'Parcela principal', 225 UNION ALL
			SELECT 302, 'demo_tenant_alpha', 3, 'Documento C', 2, 'Complemento', 75 UNION ALL
			SELECT 401, 'demo_tenant_alpha', 4, 'Documento D', 1, 'Parcela principal', 350 UNION ALL
			SELECT 402, 'demo_tenant_alpha', 4, 'Documento D', 2, 'Complemento', 50 UNION ALL
			SELECT 501, 'demo_tenant_alpha', 5, 'Documento E', 1, 'Parcela principal', 400 UNION ALL
			SELECT 502, 'demo_tenant_alpha', 5, 'Documento E', 2, 'Complemento', 100 UNION ALL
			SELECT 601, 'demo_tenant_alpha', 6, 'Documento F', 1, 'Parcela principal', 450 UNION ALL
			SELECT 602, 'demo_tenant_alpha', 6, 'Documento F', 2, 'Complemento', 150 UNION ALL
			SELECT 701, 'demo_tenant_beta', 7, 'Documento de outro tenant', 1, 'Linha de outro tenant', 90000
		)
		SELECT id, tenant_id, report_id, document_name, line_number, description, amount_cents
		FROM detail_rows`,
		IDColumn:   "id",
		Searchable: []crud.FieldKey{"description"},
	})
}

func registerEnumControlsDemo(ctx context.Context, database *sql.DB, registry *crud.Registry) error {
	// Smoke 04. O adapter infere os campos da tabela; a definição só acrescenta
	// o tamanho da página e as exceções de apresentação abaixo.
	return sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_enum_controls_demo",
		sqladapter.WithResourceKey("clear_crud_enum_controls_demo"),
		sqladapter.WithGridPageSize(10),
		// Cada WithEnum declara valores canônicos permitidos e labels traduzíveis;
		// o WithEnumControl seguinte escolhe apenas a apresentação no desktop.
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
		// Hard delete é irreversível e só aparece porque esta definição o pediu.
		// Grid e Form compartilham estes cinco campos e a ordem indicada.
		sqladapter.WithHardDelete(),
		sqladapter.WithGridColumns("name", "priority", "channel", "state", "tone"),
		sqladapter.WithFormFields("name", "priority", "channel", "state", "tone"),
	)
}
