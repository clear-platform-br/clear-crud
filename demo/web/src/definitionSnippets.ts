// These didactic excerpts mirror cmd/clear-crud-demo/definitions.go. They are
// kept in the demo so the generic renderer stays unaware of consumer source
// files and programming languages. The Go definitions remain canonical; when
// either side changes, update the corresponding excerpt and verify the smoke.
// Cada bloco explica objetivo, declaração, efeito e limites, em etapas — sem
// comentários linha a linha. Para bootstrap ou detalhes omitidos, consulte a
// função de origem indicada em docs/agent-usage.md.
export const definitionSnippets = {
  default: `// SMOKE 01 — O CRUD mínimo por convenção.
// Objetivo: publicar um recurso tenant-owned sem repetir metadados que o adapter
// consegue inferir do schema. O consumidor informa a tabela uma vez.
// O adapter descobre a chave e os campos/tipos; o clear-crud aplica a definição
// segura padrão para grade, formulário, paginação, busca, ordenação e ações.
// Use quando o comportamento convencional basta. Acrescente opções somente para
// exceções reais; o tenant ainda vem do contexto confiável do host.
// 1. Recurso: tabela física privada do tenant.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry, "clear_crud_minimal_demo_contacts"); err != nil {
  return err
}`,
  translated: `// SMOKE 02 — A mesma tabela com exceções pequenas e explícitas.
// Objetivo: provar que dois recursos podem ler a mesma tabela física com
// definições diferentes, sem duplicar tabela nem alterar seus dados.
// A chave lógica separa este CRUD dos demais; a página fica limitada a 10.
// O valor armazenado do enum continua sendo new/review/closed; Label aponta
// para mensagens traduzíveis e é somente apresentação.
// 1. Fonte e identidade pública do recurso.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_minimal_demo_contacts",
  sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_ptbr"),
  // 2. Paginação: 10 é o tamanho inicial permitido por esta definição.
  sqladapter.WithGridPageSize(10),
  // 3. Enum: valores canônicos persistidos separados de seus labels traduzidos.
  sqladapter.WithEnum("status",
    crud.Option{Value: "new", Label: "crud.status.new"},
    crud.Option{Value: "review", Label: "crud.status.review"},
    crud.Option{Value: "closed", Label: "crud.status.closed"},
  ),
); err != nil {
  return err
}`,
  defaults: `// SMOKE EXTRA 01 — Defaults estáticos de criação.
// Objetivo: mostrar o mesmo recurso com valores iniciais escolhidos pelo
// consumidor. O formulário de inclusão os apresenta e o core os aplica se a
// criação omitir esses campos; uma atualização nunca repõe defaults.
// Os defaults são escalares declarativos, não consultas nem regras de domínio.
// 1. Fonte e identidade pública do recurso.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_minimal_demo_contacts",
  sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_defaults"),
  // 2. Defaults fixos: status começa em "new" e enabled em true.
  sqladapter.WithDefault("status", "new"),
  sqladapter.WithDefault("enabled", true),
); err != nil {
  return err
}`,
  pattern: `// SMOKE EXTRA 02 — Validação declarativa de formato.
// Objetivo: exigir que name comece por letra maiúscula sem escrever callback
// no produto. O preset é uma expressão RE2 conhecida pelo clear-crud.
// A checagem acontece no servidor em criação e atualização; o campo não é
// reformatado. A mensagem é uma chave localizável, não texto de domínio fixo.
// 1. Fonte e identidade pública do recurso.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_minimal_demo_contacts",
  sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_pattern"),
  // 2. Validação server-owned e mensagem mostrada quando o formato não confere.
  sqladapter.WithPattern("name", crud.PatternFirstLetterUpper, "crud.validation.name_uppercase"),
); err != nil {
  return err
}
// 3. Sem WithGridColumns/WithFormFields: permanecem as projeções automáticas.
`,
  explicitOrder: `// SMOKE EXTRA 03 — Ordem explícita sem alterar o schema.
// Objetivo: provar que a ordem de apresentação pode ser declarada pelo
// consumidor, separadamente da ordem física das colunas e da ordem dos registros.
// Grid controla colunas da grade; Form controla campos do formulário. Como as
// listas são iguais aqui, ambos exibem status antes de name.
// 1. Fonte e identidade pública do recurso.
if err := sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_minimal_demo_contacts",
  sqladapter.WithResourceKey("clear_crud_minimal_demo_contacts_ordered"),
  // 2. Projeções ordenadas; não modifica os campos nem a ordem dos registros.
  sqladapter.WithGridColumns("status", "name", "enabled", "notes"),
  sqladapter.WithFormFields("status", "name", "enabled", "notes"),
); err != nil {
  return err
}`,
  enumControls: `// SMOKE 04 — Enums com controles de entrada escolhidos.
// Objetivo: mostrar que opções semânticas e controle visual são decisões
// separadas. Os valores persistidos são códigos; os labels são traduzíveis.
// Em desktop, cada enum usa seu controle declarado; em viewport móvel o
// renderer usa select nativo. WithHardDelete é uma política destrutiva explícita
// deste exemplo, não um default nem algo inferido da tabela.
// 1. Recurso e limite da página.
return sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_enum_controls_demo",
  sqladapter.WithGridPageSize(10),
  // 2. Prioridade: lista canônica de valores e controle select.
  sqladapter.WithEnum("priority",
    crud.Option{Value: "low", Label: "crud.priority.low"},
    crud.Option{Value: "medium", Label: "crud.priority.medium"},
    crud.Option{Value: "high", Label: "crud.priority.high"},
  ),
  sqladapter.WithEnumControl("priority", crud.EnumControlSelect),
  // 3. Canal: a mesma separação de valores/labels, com controle radio.
  sqladapter.WithEnum("channel",
    crud.Option{Value: "email", Label: "crud.channel.email"},
    crud.Option{Value: "phone", Label: "crud.channel.phone"},
    crud.Option{Value: "whatsapp", Label: "crud.channel.whatsapp"},
  ),
  sqladapter.WithEnumControl("channel", crud.EnumControlRadio),
  // 4. Estado: controle segmented no desktop.
  sqladapter.WithEnum("state",
    crud.Option{Value: "draft", Label: "crud.state.draft"},
    crud.Option{Value: "active", Label: "crud.state.active"},
    crud.Option{Value: "closed", Label: "crud.state.closed"},
  ),
  sqladapter.WithEnumControl("state", crud.EnumControlSegmented),
  // 5. Tom: controle buttons no desktop.
  sqladapter.WithEnum("tone",
    crud.Option{Value: "info", Label: "crud.tone.info"},
    crud.Option{Value: "warning", Label: "crud.tone.warning"},
    crud.Option{Value: "urgent", Label: "crud.tone.urgent"},
  ),
  sqladapter.WithEnumControl("tone", crud.EnumControlButtons),
  // 6. Hard delete é deliberado e irreversível; as projeções mantêm somente
  //    os cinco campos úteis na grade e no formulário.
  sqladapter.WithHardDelete(),
  sqladapter.WithGridColumns("name", "priority", "channel", "state", "tone"),
  sqladapter.WithFormFields("name", "priority", "channel", "state", "tone"),
)`,
  enum: `// SMOKE 03 — Formulário/grid explícitos e registros arquivados.
// Objetivo: contrastar estado de negócio (enabled) com archive (archived).
// O toggle Excluídos só consulta os arquivados porque esta definição o habilita;
// por padrão a grade mostra ativos. Linhas arquivadas ficam somente leitura.
// Os campos extras são opcionais porque este schema de demo os contém, mas o
// operador só vê as quatro colunas/campos explicitamente projetados.
// 1. Tabela física e identidade lógica deste CRUD.
return sqladapter.RegisterAutoTenantTable(
  ctx,
  database,
  registry,
  "clear_crud_disposable_junk_contacts",
  sqladapter.WithResourceKey("clear_crud_disposable_junk_contacts_enum"),
  // 2. Archive, limite de página, tamanho máximo do nome e campos opcionais.
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
  // 3. Símbolos do booleano e opções traduzíveis do enum.
  sqladapter.WithBooleanDisplay("enabled", "●", "●"),
  sqladapter.WithEnum("status",
    crud.Option{Value: "new", Label: "crud.status.new"},
    crud.Option{Value: "review", Label: "crud.status.review"},
    crud.Option{Value: "closed", Label: "crud.status.closed"},
  ),
  // 4. Lista e formulário exibem somente estes campos, nesta ordem.
  sqladapter.WithGridColumns("name", "status", "enabled", "notes"),
  sqladapter.WithFormFields("name", "status", "enabled", "notes"),
)`,
  lookup: `// SMOKE 05 — Lookups encadeados.
// Objetivo: manter IDs persistidos, mas apresentar nomes humanos vindos de
// recursos registrados. As tabelas globais são somente leitura e compartilhadas;
// os registros do formulário continuam tenant-owned.
// 1. Registre cada catálogo como recurso global permitido pelo servidor.
for _, table := range []sqladapter.Identifier{
  "clear_crud_reference_values",
  "clear_crud_reference_regions",
  "clear_crud_reference_cities",
} {
  sqladapter.RegisterAutoGlobalTable(ctx, database, registry, table)
}

// 2. CRUD consumidor: o navegador não escolhe tabelas nem colunas de lookup.
return sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_reference_lookup_demo",
  // 3. Estado não depende de outro campo e guarda id, exibindo name.
  sqladapter.WithLookup("state_id", crud.LookupDefinition{
    Resource: "clear_crud_reference_values", ValueField: "id", LabelField: "name", PageSize: 25,
  }),
  // 4. Região recebe state_id como dependência e filtra suas opções no serviço.
  sqladapter.WithLookup("region_id", crud.LookupDefinition{
    Resource: "clear_crud_reference_regions", ValueField: "id", LabelField: "name",
    Dependencies: []crud.FieldKey{"state_id"}, PageSize: 25,
  }),
  // 5. Município recebe region_id; ao trocar o ancestral, dependentes limpam.
  sqladapter.WithLookup("city_id", crud.LookupDefinition{
    Resource: "clear_crud_reference_cities", ValueField: "id", LabelField: "name",
    Dependencies: []crud.FieldKey{"region_id"}, PageSize: 25,
  }),
)`,
  fixedLookup: `// SMOKE 06 — Filtro fixo server-owned em um lookup.
// Objetivo: particionar uma tabela de referência reutilizada por várias classes
// de valor. Este trecho entra na declaração do campo state_id, dentro do
// WithLookup do CRUD consumidor; o filtro não é um parâmetro enviado pelo browser.
// Um valor em Values significa igualdade (kind = "state"); mais valores
// significam pertencimento (IN), não uma seleção feita pelo usuário.
// Dependências, escopo confiável e autorização continuam sendo aplicados junto.
// 1. Recurso de opções registrado no servidor.
sqladapter.WithLookup("state_id", crud.LookupDefinition{
  Resource: "clear_crud_reference_values",
  ValueField: "id",
  LabelField: "name",
  // 2. Restrição imutável para esta definição: moedas e outros kinds não entram.
  FixedFilters: []crud.FixedLookupFilter{
    {Field: "kind", Values: []crud.Value{"state"}},
  },
  PageSize: 25,
})`,
  auxiliaryCatalog: `// SMOKE 07 — Relação pai/filhos sem CRUD específico no core.
// Contract: clear.catalog.auxiliary.v1
// Contexto: catalogs e options já foram registrados como recursos comuns pelo
// adapter. O pai representa a tabela do catálogo; options representa seus itens.
// 1. A grade do pai lista só o nome. Ao editar, esse mesmo campo dá contexto ao
//    título do modal. Não é uma rota ou regra especial da TdT no renderer.
catalogs.Presentation.TitleField = "title"
// 2. Relação: catalog_id associa cada filho ao pai; o core carrega e muta os
//    filhos dentro da operação do pai, com limite de 99 e ações declaradas.
// 3. FieldMetadata mapeia labels/tipos/requiredness definidos pelo pai para
//    slots fixos já declarados no filho; não cria campos dinâmicos.
catalogs.Details = []crud.DetailDefinition{{
  Key: "options", Resource: options.Key, ParentField: "catalog_id",
  Maximum: 99,
  AllowCreate: true, AllowUpdate: true, AllowDelete: true,
  FieldMetadata: []crud.DetailFieldMetadataSource{
    {Field: "value_1", LabelField: "value_1_label", TypeField: "value_1_type", RequiredField: "value_1_required"},
    {Field: "value_2", LabelField: "value_2_label", TypeField: "value_2_type", RequiredField: "value_2_required"},
  },
}}`,
  auxiliaryCatalogLookup: `// SMOKE 08 — Consumidor de vários catálogos permitidos.
// Objetivo: oferecer itens ativos dos catálogos 10 (Estados) e 20
// (Territórios) em um único lookup. IDs e labels apontam para value_1/value_2.
// A lista com dois IDs é uma união intencional: qualquer item de 10 OU 20 pode
// aparecer. Se fosse apenas um catálogo, Values conteria somente aquele ID.
// O CRUD consumidor continua genérico; nenhuma tabela ou regra da TdT vai ao Vue.
// 1. Tabela física própria do consumidor e identidade server-owned.
return sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_auxiliary_catalog_lookup_demo",
  // 2. Lookup: persiste value_1 e mostra value_2 como label humano.
  sqladapter.WithLookup("selected_option", crud.LookupDefinition{
    Resource: "clear_crud_auxiliary_catalog_options",
    ValueField: "value_1",
    LabelField: "value_2",
    // 3. Filtros fixos compõem catálogo permitido com active=true.
    FixedFilters: []crud.FixedLookupFilter{
      {Field: "catalog_id", Values: []crud.Value{"10", "20"}},
      {Field: "active", Values: []crud.Value{true}},
    },
    PageSize: 25,
  }),
  // 4. Grid e formulário mostram o campo de apresentação, não o ID técnico.
  sqladapter.WithGridColumns("name", "selected_option"),
  sqladapter.WithFormFields("name", "selected_option"),
)`,
  auxiliaryCatalogChannelLookup: `// SMOKE 09 — Lookup de um único catálogo.
// Objetivo: trocar o enum Canal por um lookup no catálogo Canais (ID 30).
// Um único valor em catalog_id restringe a lista àquele grupo; active=true
// elimina opções inativas. O valor salvo é value_1 (email/phone/whatsapp) e o
// usuário vê value_2. Isto não é um seletor de catálogo/TdT.
// 1. Reutilize a tabela de registros, mas publique uma identidade CRUD própria.
return sqladapter.RegisterAutoTenantTable(ctx, database, registry,
  "clear_crud_enum_controls_demo",
  sqladapter.WithResourceKey("clear_crud_auxiliary_catalog_channel_lookup_demo"),
  // 2. O recurso de lookup já foi registrado pelo servidor; nenhum nome de
  //    tabela chega do cliente.
  sqladapter.WithLookup("channel", crud.LookupDefinition{
    Resource: "clear_crud_auxiliary_catalog_options",
    ValueField: "value_1",
    LabelField: "value_2",
    // 3. Um grupo fixo, não IN: somente canais ativos do catálogo 30.
    FixedFilters: []crud.FixedLookupFilter{
      {Field: "catalog_id", Values: []crud.Value{"30"}},
      {Field: "active", Values: []crud.Value{true}},
    },
    PageSize: 25,
  }),
  // 4. Apresente nome e canal na grade e no formulário.
  sqladapter.WithGridColumns("name", "channel"),
  sqladapter.WithFormFields("name", "channel"),
)`,
  readModel: `// SMOKE 10 — Read model com join e valores calculados.
// Objetivo: apresentar dados de múltiplas fontes sem tornar o recurso editável.
// O SELECT é registrado no servidor pelo adapter; não é recebido do browser.
// O tenant_id continua obrigatório e é aplicado pelo adapter ao resultado.
// 1. A ordem da lista define a ordem das colunas apresentadas.
columns := []crud.FieldKey{"name", "state_name", "summary", "calculated_number"}

// 2. Contrato de leitura: cada alias retornado pelo SQL tem chave e tipo
//    declarados. ReadOnly impede gravar join/cálculo em uma mutation.
definition := crud.Definition{
  Contract: crud.ContractDefinitionV1,
  Key: "clear_crud_reference_read_model_demo",
  Scope: crud.ScopeRequirements{Mode: crud.ScopeModeTenant, Keys: []string{"tenant_id"}},
  Permissions: crud.Permissions{Read: "crud.clear_crud_reference_read_model_demo.read"},
  Fields: []crud.Field{
    {Key: "name", Label: "crud.field.name", Type: crud.FieldString, Visible: true, ReadOnly: true},
    {Key: "state_name", Label: "crud.field.state_name", Type: crud.FieldString, Visible: true, ReadOnly: true},
    {Key: "summary", Label: "crud.field.summary", Type: crud.FieldString, Visible: true, ReadOnly: true},
    {Key: "calculated_number", Label: "crud.field.calculated_number", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
  },
  // 3. O grid usa a projeção acima; campos não listados não são colunas.
  Grid: crud.GridDefinition{
    Columns: columns,
  },
}

// 4. Adapter SQL: join enriquece state_name; summary concatena nomes e
//    calculated_number combina os IDs. São valores de leitura, não persistidos.
return sqladapter.RegisterReadModel(ctx, database, registry, definition,
  sqladapter.ReadModelConfig{
    Query: "SELECT c.id AS id, c.tenant_id AS tenant_id, c.name AS name, " +
      "s.name AS state_name, c.name || ' — ' || s.name AS summary, " +
      "(c.state_id * 1000) + c.region_id AS calculated_number " +
      "FROM clear_crud_reference_lookup_demo AS c " +
      "JOIN clear_crud_reference_values AS s " +
      "ON s.id = c.state_id AND s.kind = 'state'",
    // id identifica a linha; tenant_id é a chave usada para filtrar o escopo.
    IDColumn: "id",
  },
)`,
  groupedReport: `// SMOKE 12 — Relatório somente leitura com quebras em níveis.
// Objetivo: agrupar primeiro por ano e depois por mês, inserindo subtotais
// calculados no servidor entre os documentos. Vue só apresenta a resposta.
// O adapter aplica o escopo e os filtros antes da totalização; o subtotal cobre
// todos os registros correspondentes, não apenas a página visível.
// 1. Declare todos os campos usados pelo agrupamento, detalhe e totalização.
definition := crud.Definition{
  Contract: crud.ContractDefinitionV1,
  Key: "clear_crud_grouped_report_demo",
  Labels: crud.Labels{
    Title: "crud.clear_crud_grouped_report_demo.title",
    Singular: "crud.clear_crud_grouped_report_demo.singular",
  },
  Scope: crud.ScopeRequirements{Mode: crud.ScopeModeTenant, Keys: []string{"tenant_id"}},
  Permissions: crud.Permissions{Read: "crud.clear_crud_grouped_report_demo.read"},
  Fields: []crud.Field{
    // ReadOnly: o smoke representa uma fonte de relatório, não cadastro.
    {Key: "year", Label: "crud.field.year", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
    {Key: "month", Label: "crud.field.month", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
    {Key: "document_name", Label: "crud.field.document_name", Type: crud.FieldString, Visible: true, ReadOnly: true},
    {Key: "amount_cents", Label: "crud.field.amount_cents", Type: crud.FieldInteger, Visible: true, ReadOnly: true},
  },

  // 2. Grade: Columns controla linhas/colunas; Groups controla os níveis de
  //    quebra; Measures controla os subtotais. São declarações allowlisted,
  //    não expressões escolhidas pelo navegador.
  Grid: crud.GridDefinition{
    Columns: []crud.FieldKey{"year", "month", "document_name", "amount_cents"},
    Searchable: []crud.FieldKey{"document_name"},
    Sortable: []crud.FieldKey{"year", "month", "document_name", "amount_cents"},
    DefaultSort: []crud.Sort{{Field: "document_name", Direction: crud.SortAscending}},
    Pagination: crud.PaginationDefinition{
      Mode: crud.PageModeOffset, DefaultSize: 10,
      AllowedSizes: []uint16{2, 10, 25}, Total: true,
    },
    Report: &crud.ReportDefinition{
      // A ordem é hierárquica: primeiro ano, depois mês dentro daquele ano.
      Groups: []crud.ReportGroup{
        {Field: "year", Direction: crud.SortAscending},
        {Field: "month", Direction: crud.SortAscending},
      },
      // Count conta registros; Sum soma um campo inteiro, aqui centavos exatos.
      Measures: []crud.ReportMeasure{
        {Key: "documents", Label: "crud.report.documents", Operation: crud.ReportCount},
        {Key: "amount_cents", Label: "crud.report.amount_cents", Operation: crud.ReportSum, Field: "amount_cents"},
      },
    },
  },
  Presentation: crud.Presentation{Collection: crud.CollectionTable, Density: crud.DensityComfortable},
  Delete: crud.DeletePolicy{Mode: crud.DeleteModeNone},
  Concurrency: crud.ConcurrencyPolicy{Mode: crud.ConcurrencyNone},
}

// 3. Adapter: SELECT server-owned fornece linhas, datas já decompostas e
//    valores. O adapter calcula MIN/MAX dos campos numéricos no mesmo escopo,
//    e calcula os subtotais no banco. Esta versão suporta até 4 níveis e
//    8 medidas (count e sum de inteiros); não calcula média nem total geral.
//    O SQL abaixo é próprio do adapter SQLite deste demo, e seus sete registros
//    sintéticos vivem no próprio SELECT; nenhuma linha é gravada no banco.
sqladapter.RegisterReadModel(ctx, database, registry, definition, sqladapter.ReadModelConfig{
  Query: \`WITH report_rows(id, tenant_id, document_date, document_name, amount_cents) AS (
    SELECT 1, 'demo_tenant_alpha', '2025-11-01', 'Documento A', 100 UNION ALL
    SELECT 2, 'demo_tenant_alpha', '2025-11-05', 'Documento B', 250 UNION ALL
    SELECT 3, 'demo_tenant_alpha', '2025-11-20', 'Documento C', 300 UNION ALL
    SELECT 4, 'demo_tenant_alpha', '2025-12-05', 'Documento D', 400 UNION ALL
    SELECT 5, 'demo_tenant_alpha', '2026-01-03', 'Documento E', 500 UNION ALL
    SELECT 6, 'demo_tenant_alpha', '2026-01-19', 'Documento F', 600 UNION ALL
    SELECT 7, 'demo_tenant_beta', '2025-11-10', 'Documento de outro tenant', 90000
  )
  SELECT id, tenant_id,
    CAST(substr(document_date, 1, 4) AS INTEGER) AS year,
    CAST(substr(document_date, 6, 2) AS INTEGER) AS month,
    document_name, amount_cents
  FROM report_rows\`,
  IDColumn: "id",
  Searchable: []crud.FieldKey{"document_name"},
})`,
  auxiliaryDetailLookup: `// SMOKE 11 — Lookup dentro de filho detail-only.
// Objetivo: o filho não tem listagem/edição independente, mas o renderer do
// detalhe ainda resolve e mostra o label humano do código persistido.
// 1. Pai: a tabela lista apenas title e usa-o como contexto do modal.
catalogs.Presentation.TitleField = "title"

// 2. Filho: detail-only bloqueia list/get/create/update/delete diretos. O
//    lookup é metadado de apresentação/entrada usado dentro do pai; value_1 é
//    gravado e value_2 é mostrado. Os filtros restringem itens ativos do 10.
options.Access = crud.ResourceAccessDetailOnly
sqladapter.WithLookup("value_1", crud.LookupDefinition{
  Resource: options.Key,
  ValueField: "value_1",
  LabelField: "value_2",
  FixedFilters: []crud.FixedLookupFilter{
    {Field: "catalog_id", Values: []crud.Value{"10"}},
    {Field: "active", Values: []crud.Value{true}},
  },
  PageSize: 25,
})

// 3. Relação: ParentAccess permite abrir/manter detalhes somente quando o
//    campo server-owned code do pai vale 10. Isso é diferente do FixedFilters:
//    um autoriza a coleção no pai; o outro restringe as opções do lookup.
catalogs.Details = []crud.DetailDefinition{{
  Key: "options", Resource: options.Key, ParentField: "catalog_id",
  Maximum: 99, AllowCreate: true, AllowUpdate: true, AllowDelete: true,
  ParentAccess: &crud.DetailParentAccess{Field: "code", Values: []crud.Value{int64(10)}},
  // Metadados do pai dão rótulos aos slots fixos do filho; nenhum campo novo
  // é inventado em tempo de execução.
  FieldMetadata: []crud.DetailFieldMetadataSource{
    {Field: "value_1", LabelField: "value_1_label", RequiredField: "value_1_required"},
    {Field: "value_2", LabelField: "value_2_label", RequiredField: "value_2_required"},
  },
}}`,
} as const
