# Backlog do clear-crud

Este backlog registra a análise dos settings do Lazy Mofo e as capabilities
que fazem sentido para o clear-crud. A lista de referência está em
[`lazy_mofo.php`](lazy_mofo.php), um snapshot do `lazy_mofo.php` versão
2023-11-22. Ela contém 105 propriedades públicas configuráveis. O objetivo
não é reproduzir a classe: é preservar as intenções úteis em contratos
tipados, seguros e independentes do banco.

## Inventário bruto dos 105 settings

Este inventário mantém os nomes originais para que nenhum item seja perdido
durante a triagem. A decisão de tratamento está nas seções abaixo.

| Grupo | Settings do Lazy Mofo |
|---|---|
| Recurso e transporte | `$dbh`, `$table`, `$identity_name`, `$rename`, `$uri_path`, `$absolute_redirect`, `$query_string_list`, `$exclude_field` |
| Form | `$form_sql`, `$form_sql_param`, `$form_input_control`, `$form_default_value`, `$form_display_identity`, `$form_additional_html`, `$form_text_input_size` |
| Grid | `$grid_sql`, `$grid_sql_param`, `$grid_default_order_by`, `$grid_input_control`, `$grid_output_control`, `$grid_multi_delete`, `$grid_show_search_box`, `$grid_limit`, `$grid_repeat_header_at`, `$grid_show_images`, `$grid_ellipse_at`, `$grid_text_input_size` |
| Validação e ciclo de vida | `$text_input_max_length_default`, `$text_input_max_length`, `$auto_populate_controls`, `$on_insert_validate`, `$on_update_validate`, `$validate_tip_in_placeholder`, `$on_insert_user_function`, `$on_update_user_function`, `$on_delete_user_function`, `$on_update_grid_user_function`, `$after_insert_user_function`, `$after_update_user_function`, `$after_delete_user_function`, `$after_update_grid_user_function`, `$cast_user_function`, `$return_to_edit_after_insert`, `$return_to_edit_after_update`, `$redirect_using_js` |
| Locale, números e uploads | `$charset_mysql`, `$charset`, `$timezone`, `$upload_width`, `$upload_height`, `$upload_crop`, `$thumb_width`, `$thumb_height`, `$thumb_crop`, `$image_quality`, `$image_style`, `$decimal_separator`, `$restricted_numeric_input`, `$upload_allow_list` |
| Exportação e seleção | `$export_csv_file_name`, `$export_separator`, `$export_delim`, `$export_delim_escape`, `$delim`, `$select_first_option_blank` |
| Textos e UI | `$delete_confirm`, `$update_grid_confirm`, `$validate_text_general`, `$form_add_button`, `$form_update_button`, `$form_back_button`, `$form_delete_button`, `$form_text_title_add`, `$form_text_title_edit`, `$form_text_record_saved`, `$form_text_record_added`, `$grid_add_link`, `$grid_edit_link`, `$grid_delete_link`, `$grid_export_link`, `$grid_search_box`, `$grid_text_record_added`, `$grid_text_changes_saved`, `$grid_text_record_deleted`, `$grid_text_save_changes`, `$grid_text_delete`, `$grid_text_no_records_found`, `$pagination_text_use_paging`, `$pagination_text_show_all`, `$pagination_text_records`, `$pagination_text_go`, `$pagination_text_page`, `$pagination_text_of`, `$pagination_text_next`, `$pagination_text_back`, `$text_delete_image`, `$text_delete_document` |
| Storage e datas | `$upload_path`, `$thumb_path`, `$upload_path_absolute`, `$thumb_path_absolute`, `$date_out`, `$datetime_out`, `$date_in`, `$datetime_in` |

## Legenda

- `[x]` já existe no código experimental;
- `[~]` existe parcialmente e precisa de evolução;
- `[ ]` backlog;
- `[-]` não será replicado; a responsabilidade pertence ao host, ao adapter
  ou a um caso de uso próprio.

## Decisões arquiteturais

- `[x]` Renomear `Definition.List` para `Definition.Grid`.
- `[x]` Criar `Definition.Form`.
- `[x]` Manter `Definition.Fields` como catálogo único dos metadados dos
  campos; `Grid` e `Form` referenciam `FieldKey` e não duplicam descritores.
- `[x]` Fazer `Grid.Columns` controlar projeção e ordem das colunas.
- `[x]` Fazer `Form.Fields` controlar projeção e ordem do formulário.
- `[x]` Fazer `Grid.DefaultSort` controlar a ordem dos registros, distinta da
  ordem das colunas.
- `[x]` Preservar, no `AutoTable`, a ordem física das colunas quando não houver
  ordem explícita de formulário; uma projeção declarada pelo consumidor vence
  somente na visão correspondente.
- `[x]` Definir a estratégia de compatibilidade do contrato quando `List`
  virar `Grid` e `Form` for publicado: a implementação experimental usa
  `Grid`/`Form`; uma alteração incompatível futura exige nova versão pública.
- `[-]` Não aceitar `table`, coluna, tenant, adapter ou SQL vindos do HTTP.
- `[-]` Não colocar HTML, CSS ou nomes de componentes na definição.
- `[-]` Não colocar regra de domínio ou fluxo de documentos no motor genérico.

## Decisão operacional de tradução

- `[x]` Adotar o Weblate como padrão de gestão de traduções da família
  `clear*`, mantendo os catálogos versionados no Git como fonte de verdade.
- `[x]` Começar com uma única instância compartilhada em Docker/Compose; um
  projeto do Weblate representa cada produto e seus componentes representam
  frontend, backend ou documentação. A instância não precisa de servidor
  físico dedicado nesta fase.
- `[x]` Manter o Weblate fora do caminho de execução dos produtos: ele revisa,
  valida e sincroniza catálogos, mas o runtime consome os catálogos publicados
  no repositório.
- `[ ]` Fazer um piloto self-hosted com volumes persistentes do Weblate,
  PostgreSQL e Valkey, rotina de backup/restauração e permissões por projeto e
  componente.
- `[ ]` Definir layout único dos catálogos, glossário comum restrito ao
  vocabulário compartilhado e checks de CI para chaves ausentes, extras e
  placeholders incompatíveis.
- `[ ]` Reavaliar servidor dedicado somente quando escala, isolamento
  contratual, compliance ou manutenção independente justificarem a separação.

## Equivalência com `grid_sql` e `form_sql`

No Lazy Mofo, uma query podia definir campos retornados, ordem das colunas e
ordem dos registros. No clear-crud, essas decisões ficam separadas:

- `[x]` `Grid.Columns`: campos e ordem da grade;
- `[x]` `Form.Fields`: campos e ordem do formulário;
- `[x]` `Grid.DefaultSort`: ordem dos registros;
- `[x]` `Field.Visible` e `Field.ReadOnly`: exposição e edição;
- `[x]` documentar e testar projeções server-owned;
- `[~]` o contrato suporta modelos de leitura registrados por `DataSource`,
  com campos `ReadOnly` no `Grid`; o `sqladapter.RegisterReadModel` já oferece
  a primeira implementação genérica para SQL, mas os demais adapters ainda
  precisam fornecer a mesma capability;
- `[x]` permitir campos de projeção somente leitura no `Grid`, inclusive dados
  de joins usados para dar contexto ao operador; esses campos não entram na
  mutation nem precisam aparecer no `Form`;
- `[x]` `Grid.Columns`, `Grid.Searchable` e `Grid.Sortable` formam a allowlist
  server-owned; o `sqladapter.RegisterReadModel` registra a consulta no
  bootstrap sem transformar o core em parser de SQL;
- `[-]` não transportar `form_sql`, `form_sql_param`, `grid_sql` ou
  `grid_sql_param` como SQL livre na definição ou no navegador.

## Fields, defaults e validação

- `[x]` Campos obrigatórios por default; `Optional` é a exceção explícita.
- `[x]` Texto com limite default de 255 caracteres.
- `[x]` Limite específico por campo configurável na criação do CRUD.
- `[x]` Tipos string, texto, inteiro, decimal, boolean, data, datetime, email,
  telefone, enum e lookup.
- `[x]` `AutoTable` aceita enum explícito por campo, com opções e labels
  server-owned definidos pelo consumidor.
- `[x]` Expor metadados normalizados de enum/`CHECK` pela porta genérica
  `MetadataSource`; o adapter SQLite cobre `IN` e igualdades finitas no
  bootstrap, enquanto adapters de outros bancos podem implementar a mesma
  porta; labels e ordem continuam pertencendo ao CRUD.
- `[x]` Limites mínimo/máximo e comprimento mínimo/máximo.
- `[x]` Validação de email, datas, enum e tipos numéricos.
- `[x]` Campos somente leitura, sensíveis e `create_only`.
- `[x]` Defaults estáticos por campo, incluindo literais seguros obtidos do
  schema e defaults explícitos do programador; expressões dinâmicas continuam
  sob responsabilidade do adapter.
- `[x]` Aplicar default estático também a campo técnico invisível, sem publicar
  o campo no renderer nem aceitá-lo do navegador.
- `[x]` Validadores declarativos de padrão/regex com mensagens localizáveis;
  expressões RE2 server-owned são compiladas no bootstrap e aplicadas pelo
  core a valores string-backed, sem aceitar regex pelo HTTP.
- `[ ]` Transformações tipadas de entrada e normalização por campo.
- `[~]` Hooks tipados de ciclo de vida; ampliar somente quando houver caso
  genérico comprovado.
- `[-]` Não portar `on_insert_user_function`, `on_update_user_function`,
  `on_delete_user_function` e equivalentes como callbacks arbitrários.
- `[-]` Não portar `cast_user_function` livre; preferir tipos e normalizadores
  allowlisted.

## Grid

- `[x]` Renomear `ListDefinition` para `GridDefinition`.
- `[x]` Renomear `ListColumns`/referências equivalentes para `GridColumns`.
- `[x]` Limite automático de oito colunas.
- `[x]` Exceção explícita para escolher colunas da grade.
- `[x]` Busca textual server-side e filtros semânticos por coluna sem
  configuração extra: o transporte publica `Query.Filters` por
  `filter.<field>.<operator>`, o servidor valida campos/tipos e o renderer
  deriva os controles de `Grid.Columns`. Booleano envia `true`/`false`; enum
  envia o valor canônico, nunca o label traduzido. Controles enum/boolean
  aceitam múltiplas marcações e enviam `in` para a semântica “qualquer opção”.
- `[~]` Paginação offset/cursor no contrato; o core aceita ambos, mas o
  transporte HTTP/renderer atuais publicam apenas o fluxo offset.
- `[x]` Tamanho padrão da página configurável pelo consumidor, com allowlist
  ordenada e limite máximo de 100 registros.
- `[x]` Ordenação determinística e validação contra índices no adapter.
- `[x]` Regra de ordenação booleana composta com o campo seguinte.
- `[x]` Exibição de lookup na grade mostra label, nunca o ID bruto; a resolução
  ainda pode gerar várias chamadas de transporte por página.
- `[ ]` Adicionar valor de apresentação server-resolved para lookups e outros
  campos formatados, mantendo separado o valor persistido.
- `[ ]` Definir capability separada de relatório agregado: agrupamentos,
  medidas (`count`, `sum`, `avg`, `min`, `max`), subtotais e total geral, com
  allowlist server-owned, escopo/tenant, filtros e execução no adapter; não
  transformar o CRUD de linhas em um parser de `GROUP BY` recebido do cliente.
- `[ ]` Definir truncamento seguro de textos longos, sem perder acesso ao valor
  completo.
- `[ ]` Cabeçalho fixo/repetição de cabeçalho somente se a tabela realmente
  precisar de rolagem longa.
- `[ ]` Exportação CSV com allowlist de colunas, limites e auditoria.
- `[ ]` Seleção múltipla e ações em massa, somente após caso real de uso.
- `[ ]` Edição inline, somente após definir concorrência, validação e feedback.
- `[-]` Não portar `grid_input_control` como HTML/SQL livre.
- `[-]` Não portar `grid_output_control` como callbacks ou HTML arbitrário.
- `[-]` Não portar `grid_repeat_header_at` como requisito de configuração do
  consumidor; deve ser decisão do renderer quando necessário.

## Form

- `[x]` Criar `FormDefinition` com lista opcional e ordenada de `FieldKey`.
- `[x]` Formulário embutível no shell do sistema consumidor.
- `[x]` Um campo por linha, label alinhado e input alinhado.
- `[x]` Campos booleanos, enum e lookup com controles próprios.
- `[x]` Controle de enum configurável (`select`, `radio`, `segmented` ou
  `buttons`) no
  desktop; em viewport móvel o renderer força `select` nativo.
- `[x]` Lookup funciona como dropdown inicial e busca/combobox digitável; em
  mobile o controle continua responsivo sem abrir lista automaticamente no foco.
- `[x]` Ao abrir um registro existente, resolve e mostra o label atual do
  lookup sem exigir nova pesquisa do usuário.
- `[x]` Defaults estáticos visíveis no formulário de inclusão.
- `[x]` Campo opcional `Presentation.TitleField` para título contextual do
  editor Vue, com fallback para os labels da definição e sem efeito sobre
  contrato ou mutation.
- `[x]` Resolver `DetailDefinition.FieldMetadata` para slots filhos já
  declarados: label literal, tipo e requiredness vêm do pai; slot mapeado sem
  label fica oculto e não entra na mutation. O cliente HTTP preserva o mapa
  nos formatos PascalCase e camelCase antes da resolução. A convenção atende
  TdT sem acoplar o core a catálogo ou tabela.
- `[x]` Renderizar coleções mestre-detalhe como tabelas compactas, com ações por
  linha para editar, salvar, cancelar e remover; a inclusão abre somente a
  nova linha em edição e continua dentro da mutation do pai. Quando
  `ParentAccess` omite a coleção no registro, o renderer também omite a tabela
  e suas ações; coleção presente e vazia continua editável.
- `[ ]` Cabeçalho e rodapé fixos com corpo rolável para formulários longos.
- `[ ]` Seções opcionais do formulário, sem introduzir HTML na definição.
- `[-]` Não portar `form_additional_html`.
- `[-]` Não portar `form_display_identity` como identidade editável; a
  identidade técnica permanece controlada pelo adapter.
- `[-]` Não portar `form_text_input_size` como medida HTML por CRUD; usar
  tokens responsivos do renderer.

## Lookups

- `[x]` `FieldTypeLookup` e `LookupDefinition` server-owned.
- `[x]` `LookupOption` separa `Value` persistido e `Label` exibido.
- `[~]` Endpoint de lookup com busca e tamanho limitado; a continuação por
  cursor existe no core, mas ainda não é exposta pelo HTTP/renderer.
- `[x]` Escopo, autorização e limite de tamanho aplicados pelo serviço.
- `[x]` Dependências chegam pelo transporte HTTP somente como chaves
  `depends.<field>` declaradas no campo; o controller limpa dependentes
  downstream antes de nova consulta.
- `[x]` Recursos-alvo registrados são resolvidos pelo serviço; o adapter não
  recebe tabela ou coluna escolhida pelo navegador.
- `[x]` Cache read-through server-side bounded com proteção contra
  concorrência; a chave separa recurso, dependências e escopo confiável.
- `[x]` Usar uma tabela auxiliar registrada como recurso global ou tenant-owned,
  com lookup dinâmico por dependências declaradas.
- `[x]` Filtros fixos server-owned (`LookupDefinition.FixedFilters`) recebem uma
	lista tipada: um valor aplica igualdade e vários aplicam pertencimento. Eles
	se combinam com dependências, escopo, archive e cache; nunca aceitam SQL
	livre nem substituem tenant scope, autorização ou archive.
- `[~]` Escolha adaptativa entre dropdown e combobox pesquisável.
- `[ ]` Invalidação distribuída de cache quando um catálogo mutável for
  publicado; catálogos globais da primeira fatia são read-only.
- `[ ]` Retorno eficiente de labels para a grade, evitando uma consulta por
  registro.
- `[-]` Não aceitar SQL do lookup no navegador ou na definição do consumidor.
- `[-]` Não confundir lookup dinâmico com enum estático.

## Catálogos globais e dados de referência

- `[ ]` Criar modo de edição orientado a operador para catálogos auxiliares:
  ocultar metadados técnicos, mostrar somente os valores relevantes ao negócio
  e tornar o mestre-detalhe compreensível para usuários finais; manter o modo
  técnico completo para desenvolvedores e agentes. Os títulos contextuais do
  renderer já estão disponíveis, mas não encerram esta melhoria de UX.
- `[x]` Modo explícito `global` para recursos de referência somente leitura.
- `[x]` `RegisterAutoGlobalTable` sem `tenant_id` duplicado por tenant.
- `[x]` Recurso global pode ser alvo de lookup de qualquer CRUD autorizado.
- `[x]` Cache compartilhável por chave de recurso, dependências e escopo.
- `[x]` Exclusão física somente por exceção explícita `WithHardDelete()`;
  ausência da opção não oferece ação de exclusão.
- `[ ]` Exemplo de importação versionada para uma fonte externa (FIPE,
  Correios, companhia aérea etc.); o importador permanece fora do motor.
- `[ ]` Associação opcional `tenant_id -> reference_id` para catálogos
  globais restringidos por tenant.
- `[-]` Não embutir estados, municípios, marcas, modelos, aeroportos ou regras
  de uma fonte comercial no core.

## Anexos, imagens e documentos simples

- `[~]` Manter a proposta em [`assets-proposal.md`](assets-proposal.md).
- `[ ]` Definir capability opt-in para imagem, documento e arquivo.
- `[ ]` Repositório nomeado/registrado no servidor, nunca path ou bucket vindo
  do cliente.
- `[ ]` Allowlist de MIME e extensão, com verificação do conteúdo real.
- `[ ]` Limite de bytes por arquivo e limite de quantidade por registro.
- `[ ]` Limites de dimensões, miniaturas e qualidade para imagens.
- `[ ]` Nome/chave gerados pelo servidor; nunca confiar no nome ou caminho
  enviado pelo navegador.
- `[ ]` Metadados e referência no banco; bytes em storage protegido.
- `[ ]` URLs temporárias/signed URLs para leitura privada.
- `[ ]` Formulário para incluir, substituir e remover anexos.
- `[ ]` Grid com miniatura opcional e preview/download seguro.
- `[ ]` Política explícita para archive, hard delete, substituição e limpeza
  de objetos órfãos.
- `[ ]` Auditoria das mutações de anexos.
- `[-]` Não portar `$upload_path`, `$thumb_path` ou filesystem local como parte
  do contrato público.
- `[-]` Não incluir OCR, assinatura, aprovação, processamento documental ou
  workflow no CRUD simples.

## Locale, números e datas

- `[x]` Valores canônicos para data, datetime e decimal no contrato.
- `[x]` Mensagens localizáveis no renderer.
- `[ ]` Formatação localizada de datas, horas e números apenas na apresentação.
- `[ ]` Definir origem confiável do timezone no host.
- `[-]` Não portar `$charset_mysql` ou configuração específica de MySQL para o
  core.
- `[-]` Não usar parsing permissivo de datas/números como substituto da
  validação canônica do backend.

## Exportação e valores múltiplos

- `[ ]` Exportação CSV server-side, com autorização, allowlist e limite de
  volume.
- `[ ]` Definir formato e escape sem depender da configuração do navegador.
- `[ ]` Campo multivalor somente se houver necessidade real de domínio.
- `[ ]` Seleção opcional em lookup com cardinalidade e validação explícitas.
- `[-]` Não portar `$delim` como convenção genérica para armazenar listas em
  uma string sem contrato de tipo.

## Mensagens, i18n e acessibilidade

- `[x]` Catálogo comum `ptBR`/`enUS`.
- `[x]` Labels e ajuda resolvidos por `MessageCode`.
- `[x]` Rótulos acessíveis para ações e booleanos.
- `[x]` Mensagens específicas de lookup no renderer; mensagens de anexos e
  exportação aguardam essas capabilities.
- `[ ]` Testes de teclado, foco, leitor de tela e erros de campo para todos os
  novos controles.
- `[ ]` Após encerrar os settings planejados, executar uma fatia dedicada de
  operabilidade do renderer por teclado e mouse: ordem de foco, Tab,
  setas para navegar opções de combobox/lookup, Enter, Escape e feedback de
  foco. O caso observado hoje exige clique do mouse para selecionar uma opção
  filtrada de lookup; não corrigir pontualmente antes dessa revisão integral.
- `[-]` Não permitir que o consumidor substitua botões, links ou busca por
  strings HTML arbitrárias.

## Roteamento, transporte e host

- `[x]` Renderer sempre embutível no shell do produto.
- `[x]` Preferências de tema e densidade locais ao recurso.
- `[-]` Não portar `$uri_path`, `$query_string_list` ou `$absolute_redirect`;
  rota e navegação pertencem ao host.
- `[-]` Não portar `$redirect_using_js`; usar transporte e router do host.
- `[x]` Documentar integração: o renderer é sempre embutível e Header, Menu,
  Sidebar e Footer pertencem ao host.

## Segurança e operação

- `[x]` Escopo confiável obrigatório antes de consultas e mutações.
- `[x]` Tenant nunca escolhido pelo navegador.
- `[x]` Chaves de escopo server-owned adicionais podem particionar recursos e
  read models por `Definition.Scope.Keys`/`ScopeColumns`; o `ScopeProvider`
  fornece os valores e o navegador não os escolhe. Isso não substitui filtros
  opcionais de grade nem transforma política do pai em status dos filhos.
- `[x]` Modo genérico `ResourceAccessDetailOnly` com `DetailParentAccess`: bloqueia
  list/get/mutations independentes, mas mantém `Definition` e lookups declarados
  como metadados seguros para o renderer resolver labels no detalhe. A edição
  transacional dentro do pai só ocorre quando o campo server-owned do pai contém
  um valor permitido; registros sem esse valor não recebem a coleção no
  renderer. Não há regra de catálogo, filtro local ou endpoint paralelo.
- `[ ]` Predicado fixo server-owned para manter uma rota independente do filho e,
  simultaneamente, consultar a política do pai em list/get/create/update/delete;
  abrir somente se um caso genérico real exigir esse fluxo além de `detail-only`.
- `[x]` Permissões revalidadas no backend.
- `[x]` Archive e hard delete com política explícita.
- `[x]` Visibilidade declarada de archive no Grid: manter `active_only` por
  default e permitir uma projeção server-owned `active_and_archived`, com
  toggle curto `Excluídos` e nome acessível/tooltip `Incluir registros
  arquivados`; sem parâmetro livre do navegador. Registros arquivados começam
  como somente leitura e restauração será capability própria.
- `[ ]` Restauração de registros `archive`: capability pública separada
  (`ActionRestore` e permissão declarada pelo consumidor), porta genérica
  `DataSource.Restore` executada pelo adapter, endpoint versionado e controle
  por `Authorizer`/role do host. Exigir confirmação, versão otimista, exibição
  explícita de `Excluído`, ação `Restaurar` somente para papel autorizado e
  auditoria transacional da exclusão, restauração, conflitos e negativas; sem
  mudança de schema no core. `hard_delete` continua irrecuperável.
- `[x]` Concorrência otimista e auditoria de mutações.
- `[~]` Limites de tamanho e volume já existem para body HTTP, busca, página,
  cursor e lookup; faltam orçamento/timeouts formais e limites de exportação e
  anexos.
- `[ ]` Telemetria de leitura sem bloquear o CRUD.
- `[ ]` Suíte de conformidade para lookup, projeções e anexos quando publicados.

## Escala e concorrência

Este grupo é um critério transversal: qualquer capability nova deve ser
avaliada para milhares de usuários concorrentes antes de ser considerada
pronta para produção.

- `[x]` Manter limites de página, busca e payload; nunca aceitar consultas ou
  resultados sem limite operacional.
- `[x]` Manter o tenant no escopo server-owned de todas as leituras e
  mutações, com índices compatíveis no adapter.
- `[ ]` Definir a política de totalização da paginação: evitar `COUNT(*)`
  repetido para a mesma consulta/tenant, com cache seguro, TTL e invalidação
  após mutações quando a exatidão imediata não for obrigatória.
- `[ ]` Definir timeouts, orçamento de consulta e comportamento sob saturação
  para listagem, busca, lookup, exportação e anexos.
- `[ ]` Criar testes de carga e concorrência para paginação, busca, mutações e
  isolamento entre tenants; testes unitários/race não substituem esses testes.
- `[ ]` Revisar novas queries para N+1, joins sem índices, locks longos e
  contagens completas desnecessárias antes de incorporá-las ao adapter.

## Critério de encerramento

Uma capability só sai deste backlog quando houver:

1. contrato público tipado;
2. validação server-side;
3. comportamento do adapter sem vazamento de escopo;
4. renderer acessível e responsivo;
5. testes unitários, integração e, quando aplicável, race/vet/conformance;
6. documentação de uso e decisão sobre o que permanece fora do CRUD genérico;
7. mapa de capabilities, catálogo de settings e manual de agentes atualizados
   na mesma alteração.
