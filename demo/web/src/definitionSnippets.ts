// These readable snippets mirror cmd/clear-crud-demo/definitions.go. They are
// intentionally kept in the demo so the generic CRUD renderer stays unaware
// of consumer source files and programming languages.
// Os comentários numerados seguem a mesma receita: recurso, regras,
// relações ou lookups, projeções e registro.
export const definitionSnippets = {
  default: `// 1. Tabela física: o adapter infere o contrato básico do schema.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts"); err != nil {
  return err
}`,
  translated: `// 1. Tabela física e chave pública do recurso.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_minimal_demo_contacts",
  sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_ptbr"),
  // 2. Paginação e opções traduzidas do campo status.
  sqladapter.WithGridPageSize(10),
  sqladapter.WithEnum("status",
    crud.Option{Value: "new", Label: "crud.status.new"},
    crud.Option{Value: "review", Label: "crud.status.review"},
    crud.Option{Value: "closed", Label: "crud.status.closed"},
  ),
); err != nil {
  return err
}`,
  defaults: `// 1. Tabela física e chave pública do recurso.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_minimal_demo_contacts",
  sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_defaults"),
  // 2. Defaults aplicados pelo servidor na criação.
  sqladapter.WithDefault("status", "new"),
  sqladapter.WithDefault("enabled", true),
); err != nil {
  return err
}`,
  pattern: `// 1. Tabela física e chave pública do recurso.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_minimal_demo_contacts",
  sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_pattern"),
  // 2. Validação declarativa com preset: o nome começa com letra maiúscula.
  sqladapter.WithPattern("name", crud.PatternFirstLetterUpper, "crud.validation.name_uppercase"),
); err != nil {
  return err
}
// 3. Sem projeção explícita: grid e formulário usam os defaults.
`,
  explicitOrder: `// 1. Tabela física e chave pública do recurso.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_minimal_demo_contacts",
  sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_ordered"),
  // 2. Projeções explícitas para grid e formulário.
  sqladapter.WithGridColumns("status", "name", "enabled", "notes"),
  sqladapter.WithFormFields("status", "name", "enabled", "notes"),
); err != nil {
  return err
}`,
  enumControls: `// 1. Tabela física e tamanho da página.
return sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_enum_controls_demo",
  sqladapter.WithGridPageSize(10),
  // 2. Opções e controle visual de prioridade.
  sqladapter.WithEnum("priority",
    crud.Option{Value: "low", Label: "crud.priority.low"},
    crud.Option{Value: "medium", Label: "crud.priority.medium"},
    crud.Option{Value: "high", Label: "crud.priority.high"},
  ),
  sqladapter.WithEnumControl("priority", crud.EnumControlSelect),
  // 3. Opções e controle visual de canal.
  sqladapter.WithEnum("channel",
    crud.Option{Value: "email", Label: "crud.channel.email"},
    crud.Option{Value: "phone", Label: "crud.channel.phone"},
    crud.Option{Value: "whatsapp", Label: "crud.channel.whatsapp"},
  ),
  sqladapter.WithEnumControl("channel", crud.EnumControlRadio),
  // 4. Opções e controle visual de estado.
  sqladapter.WithEnum("state",
    crud.Option{Value: "draft", Label: "crud.state.draft"},
    crud.Option{Value: "active", Label: "crud.state.active"},
    crud.Option{Value: "closed", Label: "crud.state.closed"},
  ),
  sqladapter.WithEnumControl("state", crud.EnumControlSegmented),
  // 5. Opções e controle visual de tom.
  sqladapter.WithEnum("tone",
    crud.Option{Value: "info", Label: "crud.tone.info"},
    crud.Option{Value: "warning", Label: "crud.tone.warning"},
    crud.Option{Value: "urgent", Label: "crud.tone.urgent"},
  ),
  sqladapter.WithEnumControl("tone", crud.EnumControlButtons),
  // 6. Exclusão física e projeções do grid e do formulário.
  // A exclusão física só aparece porque esta linha foi declarada explicitamente.
  sqladapter.WithHardDelete(),
  sqladapter.WithGridColumns("name", "priority", "channel", "state", "tone"),
  sqladapter.WithFormFields("name", "priority", "channel", "state", "tone"),
)`,
  enum: `// 1. Tabela física e chave pública do recurso.
return sqladapter.RegisterAutoTenantTable(
  ctx,
  database,
  registry,
  "clear_crud_disposable_junk_contacts",
  sqladapter.WithResourceKey("clear_crud_disposable_junk_contacts_enum"),
  // 2. Arquivamento, paginação e regras de entrada.
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
  // 3. Apresentação do booleano e opções do enum.
  sqladapter.WithBooleanDisplay("enabled", "●", "●"),
  sqladapter.WithEnum("status",
    crud.Option{Value: "new", Label: "crud.status.new"},
    crud.Option{Value: "review", Label: "crud.status.review"},
    crud.Option{Value: "closed", Label: "crud.status.closed"},
  ),
  // 4. Projeções explícitas do grid e do formulário.
  sqladapter.WithGridColumns("name", "status", "enabled", "notes"),
  sqladapter.WithFormFields("name", "status", "enabled", "notes"),
)`,
  lookup: `// 1. Registre as tabelas que fornecem as opções dos lookups.
for _, table := range []sqladapter.Identifier{
  "clear_crud_reference_values",
  "clear_crud_reference_regions",
  "clear_crud_reference_cities",
} {
  sqladapter.RegisterAutoGlobalTable(ctx, database, registry, table)
}

  // 2. Tabela do CRUD consumidor.
return sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_reference_lookup_demo",
  // 3. Lookup direto de estado.
  sqladapter.WithLookup("state_id", crud.LookupDefinition{
    Resource: "clear_crud_reference_values", ValueField: "id", LabelField: "name", PageSize: 25,
  }),
  // 4. Lookup dependente do estado.
  sqladapter.WithLookup("region_id", crud.LookupDefinition{
    Resource: "clear_crud_reference_regions", ValueField: "id", LabelField: "name",
    Dependencies: []crud.FieldKey{"state_id"}, PageSize: 25,
  }),
  // 5. Lookup dependente da região.
  sqladapter.WithLookup("city_id", crud.LookupDefinition{
    Resource: "clear_crud_reference_cities", ValueField: "id", LabelField: "name",
    Dependencies: []crud.FieldKey{"region_id"}, PageSize: 25,
  }),
)`,
  fixedLookup: `// 1. Reutilize a tabela de opções como fonte do lookup.
sqladapter.WithLookup("state_id", crud.LookupDefinition{
  Resource: "clear_crud_reference_values",
  ValueField: "id",
  LabelField: "name",
  // 2. Restrinja o resultado a uma categoria fixa do catálogo.
  FixedFilters: []crud.FixedLookupFilter{
    {Field: "kind", Values: []crud.Value{"state"}},
  },
  PageSize: 25,
})`,
  auxiliaryCatalog: `// 1. Contrato público do catálogo auxiliar.
// Contract: clear.catalog.auxiliary.v1
// 2. Tabela pai: o grid mostra somente o nome do catálogo.
// 3. Tabela filha: options guarda os itens e recebe o vínculo pelo catalog_id.
catalogs.Details = []crud.DetailDefinition{{
  Key: "options", Resource: options.Key, ParentField: "catalog_id",
  Maximum: 99, AllowCreate: true, AllowUpdate: true, AllowDelete: true,
}}`,
  auxiliaryCatalogLookup: `// 1. Tabela física e recurso consumidor.
return sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_auxiliary_catalog_lookup_demo",
  // 2. Lookup nos itens da TdT.
  sqladapter.WithLookup("selected_option", crud.LookupDefinition{
    Resource: "clear_crud_auxiliary_catalog_options",
    ValueField: "value_1",
    LabelField: "value_2",
    // 3. Restrinja aos catálogos permitidos e aos itens ativos.
    FixedFilters: []crud.FixedLookupFilter{
      {Field: "catalog_id", Values: []crud.Value{"10", "20"}},
      {Field: "active", Values: []crud.Value{true}},
    },
    PageSize: 25,
  }),
  // 4. Projeções do grid e do formulário.
  sqladapter.WithGridColumns("name", "selected_option"),
  sqladapter.WithFormFields("name", "selected_option"),
)`,
  auxiliaryCatalogChannelLookup: `// 1. Recurso consumidor e chave pública.
return sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_enum_controls_demo",
  sqladapter.WithResourceKey("clear_crud_auxiliary_catalog_channel_lookup_demo"),
  // 2. Lookup de canais na tabela filha da TdT.
  sqladapter.WithLookup("channel", crud.LookupDefinition{
    Resource: "clear_crud_auxiliary_catalog_options",
    ValueField: "value_1",
    LabelField: "value_2",
    // 3. Catálogo 30 e somente itens ativos.
    FixedFilters: []crud.FixedLookupFilter{
      {Field: "catalog_id", Values: []crud.Value{"30"}},
      {Field: "active", Values: []crud.Value{true}},
    },
    PageSize: 25,
  }),
  // 4. Projeções do grid e do formulário.
  sqladapter.WithGridColumns("name", "channel"),
  sqladapter.WithFormFields("name", "channel"),
)`,
  readModel: `// 1. Projeção ordenada do grid.
columns := []crud.FieldKey{"name", "state_name", "summary", "calculated_number"}

// 2. Contrato, escopo e campos somente leitura.
definition := crud.Definition{
  Key: "clear_crud_reference_read_model_demo",
  Scope: crud.ScopeRequirements{Mode: crud.ScopeModeTenant, Keys: []string{"tenant_id"}},
  Permissions: crud.Permissions{Read: "crud.clear_crud_reference_read_model_demo.read"},
  Fields: []crud.Field{
    {Key: "name", Label: "crud.field.name", Type: crud.FieldString, Visible: true, ReadOnly: true},
    {Key: "state_name", Label: "crud.field.state_name", Type: crud.FieldString, Visible: true, ReadOnly: true},
    {Key: "summary", Label: "crud.field.summary", Type: crud.FieldString, Visible: true, ReadOnly: true},
    {Key: "calculated_number", Label: "crud.field.calculated_number", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
  },
  // 3. O grid usa a projeção declarada acima.
  Grid: crud.GridDefinition{
    Columns: columns,
  },
}

// 4. Consulta server-owned: join e campo calculado são preparados no adapter.
return sqladapter.RegisterReadModel(ctx, database, registry, definition,
  sqladapter.ReadModelConfig{
    Query: "SELECT c.id AS id, c.tenant_id AS tenant_id, c.name AS name, " +
      "s.name AS state_name, c.name || ' — ' || s.name AS summary, " +
      "(c.state_id * 1000) + c.region_id AS calculated_number " +
      "FROM clear_crud_reference_lookup_demo AS c " +
      "JOIN clear_crud_reference_values AS s " +
      "ON s.id = c.state_id AND s.kind = 'state'",
    IDColumn: "id",
  },
)`,
} as const
