# Mapa atual de capabilities

Este documento é a fotografia do que o repositório oferece hoje. Ele
complementa o backlog: `x` significa comportamento implementado e testado,
`~` significa que existe no core, mas há uma limitação no adapter, transporte
ou renderer, e `-` significa que ainda não deve ser usado.

## Disponível

| Grupo | Situação atual |
| --- | --- |
| Registro | Definições server-owned, validação no bootstrap, cópia defensiva e registry lacrado. |
| Escopo | Recursos tenant-scoped ou global explícitos; chaves adicionais server-owned podem particionar uma visão via `Definition.Scope.Keys`/`ScopeColumns`; o navegador nunca escolhe tenant nem escopo confiável. |
| Autorização | Permissão por ação e revalidação backend antes de leitura e mutação. |
| Campos | Tipos escalares, required por default, `Optional`, defaults estáticos de criação inclusive em campos técnicos invisíveis, limites de tamanho/faixa, sensibilidade, somente leitura, `create_only`, padrões RE2 declarativos com mensagens localizáveis e presets públicos `crud.Pattern*` para formatos recorrentes. |
| Grid | Projeção/ordem por `Grid.Columns`, busca textual global server-side e filtros automáticos por coluna via `Query.Filters`, ordenação determinística, paginação offset no HTTP atual e limite máximo de 100; no `AutoTable`, a ordem física do banco é preservada quando não há projeção explícita. Archive permanece `active_only` por default; `Grid.ArchiveVisibility` pode habilitar o toggle compacto `Excluídos`, com linhas arquivadas somente para leitura, selo `Excluído` na coluna de ações e faixa visual dedicada. |
| Form | `Form.Fields`, um campo por linha, defaults estáticos visíveis na inclusão, validação local e server-side, enum, booleano, lookup e resolução de metadados de campos filhos fixos por valores do pai. |
| Mutação | Create/update/delete versionados, `UnitOfWork`, auditoria obrigatória e hooks tipados. |
| Exclusão | Nenhuma ação por default; archive e hard delete somente por declaração explícita. |
| Mestre-detalhe | Uma coleção filha direta, com cardinalidade, mutation transacional e `DetailFieldMetadataSource` para label/tipo/requiredness server-owned de slots já declarados; slot mapeado sem label fica oculto; sem árvore ou netos. |
| Lookup | Valor separado de label, dependências dinâmicas, filtros fixos server-owned por lista tipada (igualdade ou pertencimento), busca, primeira página limitada, cache bounded (512 entradas/5 min por default) e resolução do label na grade. |
| Catálogo global | `RegisterAutoGlobalTable` para referência compartilhada, read-only e sem duplicação por tenant. |
| Metadados | `MetadataSource` promove `CHECK`/enum finito para opções estruturais e pode marcar campos de modelos de leitura como somente leitura durante o bootstrap. |
| Vue | Renderer embutível, responsivo, pt-BR/en-US, tema claro/escuro, densidade, controles de enum e título contextual opcional via `Presentation.TitleField`, com fallback server-owned. |
| Tradução | `MessageCode` estável e catálogos fornecidos pelo consumidor; Weblate foi escolhido como fluxo externo de gestão, revisão e sincronização, sem dependência no runtime. |
| Operação | `AuditSink`, correlation ID, erros públicos sanitizados, testes, race, vet, cobertura e govulncheck no gate. |

## Parcial ou com limite conhecido

### Transporte HTTP

O core aceita paginação offset e cursor, mas o `httpadapter` e o client Vue
atuais expõem a listagem por `page`/`size`; o cursor ainda não está publicado
como fluxo completo no renderer. O endpoint de lookup aceita busca e tamanho,
mas o HTTP atual devolve apenas as opções, sem expor a continuação do cursor.

O core possui filtros allowlisted (`eq`, `ne`, comparadores, `contains`, `prefix`,
`is_null` e `in`) para chamadas de `Service`/`DataSource`. O HTTP publica a mesma
capability por chaves `filter.<campo>.<operador>`, sempre contra campos
server-owned da definição e sem aceitar coluna ou SQL do navegador. O renderer
monta o painel automaticamente a partir das colunas públicas de `Grid.Columns`.
Para enum/boolean, múltiplas marcações usam `filter.<campo>.in` repetido; o
adapter traduz isso para pertencimento parametrizado.

O campo `Buscar registros` continua sendo textual e não substitui um filtro
semântico por coluna. Para um enum, ele pesquisa o valor persistido quando o
adapter o encontra, mas não conhece o label traduzido da tela; para booleanos,
o painel envia `true`/`false` de forma tipada. Assim os labels são apenas
apresentação e os valores canônicos permanecem independentes do idioma.

### Labels de lookup na grade

O grid não exibe o ID cru: resolve o label e mostra `—` enquanto o valor está
pendente ou indisponível. A implementação ainda pode gerar várias chamadas de
lookup por página; o cache e o single-flight protegem o banco, mas a redução de
chamadas no transporte continua backlog.

### Busca de lookup

O renderer exige três caracteres por padrão antes de buscar texto e permite
`LookupDefinition.MinSearchLength` como exceção. A seta abre a página inicial
limitada. Essa regra pertence ao renderer; clients próprios devem respeitar o
metadado e o serviço continua impondo limites máximos de consulta.

### Tabela auxiliar única

Tabelas auxiliares registradas como recursos globais ou tenant-owned podem ser
particionadas por `LookupDefinition.FixedFilters`, com uma lista tipada
declarada no bootstrap, como `Values: []crud.Value{"estado"}`. Um valor aplica
igualdade e vários aplicam pertencimento. Eles se somam a dependências, escopo
confiável e archive; não são parâmetros HTTP nem substituem autorização.

Uma política de manutenção de catálogo é metadado do recurso pai, não status
de cada opção filha. O consumidor pode usar um escopo confiável adicional para
uma visão operacional que sempre deva ser particionada, mas isso exige valor do
`ScopeProvider`; não é um filtro livre do navegador. Os nomes e valores da
política (`system`, `customizable`, `user`, por exemplo) continuam pertencendo
ao produto consumidor.

### Modelos de leitura registrados

O contrato já aceita campos server-owned `ReadOnly` retornados pelo `DataSource`,
inclusive valores vindos de join ou cálculo, e permite colocá-los na allowlist
de `Grid.Columns` sem aceitá-los em mutações. O `sqladapter` oferece
`RegisterReadModel`, que valida a consulta server-owned no bootstrap, aplica
escopo, filtros, busca e paginação e marca os campos retornados como somente
leitura. A consulta continua pertencendo ao adapter; ela não vira SQL no core
nem parâmetro do navegador. Outros adapters devem fornecer a mesma capability
sem alterar a definição do consumidor.

## Ainda não disponível

- equivalência de modelos de leitura registrados nos adapters PostgreSQL,
  MySQL/MariaDB e demais adapters;
- relatórios agregados com agrupamento, medidas, subtotais e total geral;
- validadores declarativos de regex e transformações tipadas;
- exportação CSV, seleção em massa e edição inline;
- anexos, imagens, documentos, miniaturas e URLs temporárias;
- formatação localizada de datas, horas e números;
- invalidação distribuída de cache e orçamento/timeouts de consulta;
- testes de carga e política formal para evitar `COUNT(*)` repetido;
- adapters PostgreSQL, MySQL e MariaDB.
- piloto operacional do Weblate self-hosted, layout único de catálogos, checks
  de CI e rotina de backup/restauração.

## Rotas HTTP atuais

| Método | Rota | Uso |
| --- | --- | --- |
| `GET` | `/api/v1/crud/{resource}/definition` | definição pública filtrada por autorização |
| `GET` | `/api/v1/crud/{resource}/records?q=&filter.<field>.<operator>=&page=&size=&include_archived=` | grid paginado por offset; filtros usam campos server-owned e operadores allowlisted; `include_archived=true` só funciona quando o Grid declara `active_and_archived` |
| `POST` | `/api/v1/crud/{resource}/records` | criação |
| `GET` | `/api/v1/crud/{resource}/records/{id}` | leitura individual |
| `PUT` | `/api/v1/crud/{resource}/records/{id}` | atualização com `version` |
| `DELETE` | `/api/v1/crud/{resource}/records/{id}` | archive ou hard delete declarado |
| `GET` | `/api/v1/crud/{resource}/lookups/{field}` | lookup com `q`, `size` e `depends.<field>` |

O handler nunca aceita tabela, coluna, SQL, adapter ou tenant na entrada. Erros
públicos usam `401`, `403`, `404`, `409`, `422`, `429` ou `503`, conforme o
código sanitizado.

## Regra de manutenção documental

Toda capability nova deve atualizar, na mesma alteração:

1. este mapa, com situação e limitação;
2. [`backlog.md`](backlog.md), com o item e seu critério de encerramento;
3. [`settings-catalog.json`](settings-catalog.json), quando houver setting;
4. [`agent-usage.md`](agent-usage.md), com decisão de uso e fronteiras;
5. cenários e testes correspondentes.
