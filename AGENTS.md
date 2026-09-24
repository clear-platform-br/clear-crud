# AGENTS.md — clear-crud

## Propósito

`clear-crud` implementa a capability pública `clear.crud`: CRUDs administrativos
simples orientados por definição, sem domínio, banco, tenant ou frontend próprios.

O contrato canônico é governado no repositório `clear_platform` pelos IDs:

- `clear.crud.definition.v1`;
- `clear.crud.datasource.v1`;
- `clear.crud.http.v1`;
- `clear.crud.renderer.v1`.

## Limites

- O core depende apenas da biblioteca padrão do Go.
- Não usar ORM, driver SQL, router HTTP ou framework de frontend no pacote raiz.
- Dados, migrations, credenciais, tenants, autorização real e auditoria
  persistida pertencem ao produto consumidor.
- Nenhuma entrada HTTP pode escolher tenant, tabela, coluna, SQL ou adapter.
- Não copiar contratos ou código de produtos; consumidores usam releases públicas.
- Fluxos com documentos, lotes, aprovações ou efeitos externos permanecem casos
  de uso próprios, fora do CRUD simples.

## Desenvolvimento

- API pública usa identificadores e comentários GoDoc em inglês.
- Documentação durável e cenários operacionais usam pt-BR.
- Antes de concluir alteração Go, executar `gofmt`, `go test ./...`,
  `go test -race ./...` e `go vet ./...`.
- `make validate` é o gate de regressão obrigatório: build, testes unitários,
  race detector, vet, cobertura e govulncheck. A cobertura global não pode
  cair de 90%; o alvo contínuo é 95%, sem substituir testes explícitos dos
  ramos críticos.
- Cada PR deve alterar uma fatia pequena e manter os contratos existentes
  compatíveis ou registrar nova major com guia de migração.
- Adapters executam a suíte de conformidade; mocks não substituem banco real.
