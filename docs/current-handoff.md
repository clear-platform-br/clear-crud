# Handover atual — clear-crud

Atualizado em 2026-09-28.

## Objetivo imediato

Continuar a evolução genérica do `clear-crud` sem acoplar o motor a domínio,
banco, tenant ou frontend de produto. A primeira fatia de filtros fixos de
lookup foi concluída com filtros fixos server-owned por lista tipada: um valor
aplica igualdade e vários aplicam pertencimento; a próxima recomendada é a
suíte de conformidade e a decisão por um adapter adicional baseada em
consumidor real.

## Estado atual

- Core Go genérico com `Definition`, `DataSource`, `Service`, escopo confiável,
  autorização, auditoria transacional, validação, paginação e archive.
- `sqladapter` SQLite funcional, incluindo `AutoTable`, introspecção segura,
  enums/checks, lookups dependentes, filtros fixos por lista tipada e cache bounded
  de lookup.
- Renderer Vue padrão embutível, com grid, form, dark mode, densidade,
  enum controls, lookup label resolution e mensagens de busca mínima.
- Documentação principal: `docs/capability-map.md`, `docs/backlog.md`,
  `docs/settings-catalog.json` e `docs/agent-usage.md`.
- Última validação completa conhecida: `make validate` passou; cobertura global
  em 90% e govulncheck sem vulnerabilidades.

## Decisões que não podem regredir

- O CRUD é sempre conteúdo embutível; o produto consumidor possui shell,
  header, menu, sidebar, footer, tema global e autenticação.
- Adapters traduzem contratos; não decidem comportamento de produto.
- Nenhum SQL, tabela, coluna, tenant ou adapter pode ser escolhido pela entrada
  HTTP. Escopo confiável é aplicado antes das consultas e mutações.
- Registros soft-deleted ficam fora dos caminhos normais; hard delete é uma
  política explícita e a integridade referencial é responsabilidade do banco.
- Campos são obrigatórios por padrão; `Optional` é exceção declarada.
- Lookups retornam sempre `value + label`; o grid não pode exibir IDs no lugar
  dos labels.
- Dados globais de referência podem ser registrados sem tenant, explicitamente,
  como recursos globais somente leitura.
- Não armazenar blobs no banco: anexos devem ser uma capacidade opt-in baseada
  em metadados/referências e storage externo protegido.

## Próximas fatias

1. Adapters adicionais: primeiro suíte de conformidade e decisão baseada em
   consumidor real; não adicionar drivers ao pacote raiz por especulação.
2. Joins/read models: projeção/read model registrada no servidor, sem SQL vindo
   do navegador e sem transformar o CRUD em ORM.
3. Anexos: contrato neutro de metadata + referência + limites + storage adapter;
   nunca bytes ricos no PostgreSQL/SQLite do CRUD.
4. Testes de carga: cenários de list/lookup/mutação, concorrência, p95/p99,
   contagem de queries, cache e isolamento de escopo. Não extrapolar SQLite para
   capacidade de produção de outro banco.

## Validação obrigatória após alterações Go

```sh
gofmt -w <arquivos-go>
go test ./...
go test -race ./...
go vet ./...
make validate
```

Antes de editar, conferir `pwd`, `git status --short` e alterações já existentes.
