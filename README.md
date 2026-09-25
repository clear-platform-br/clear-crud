# Clear CRUD

`clear-crud` é a implementação pública da capability `clear.crud`: cadastros
administrativos simples, orientados por definição e sem acoplamento a domínio,
banco ou frontend.

O módulo está em sua primeira release pública (`v0.1.0`). O primeiro piloto
será executado no Aztheca Lab, mas o módulo não conhece condomínio, boleto,
contato ou qualquer vocabulário do laboratório.

Agentes e consumidores devem seguir o [manual de uso](docs/agent-usage.md)
antes de iniciar um CRUD: ele define o critério objetivo para usar o motor ou
escrever um caso de uso próprio, sem duplicar a capability no produto.

## O que este módulo fará

- registrar recursos fechados no servidor;
- validar uma definição antes de o recurso ficar disponível;
- aplicar escopo, autorização, validação, concorrência e auditoria por ports;
- executar CRUD simples por adapters SQL ou não relacionais conformes;
- publicar contratos HTTP e renderers opcionais desacoplados.

Para uma tabela que use as convenções suportadas, o consumidor poderá declarar
campos, permissões e regras de domínio sem reescrever repository, handlers,
busca, paginação, formulário ou feedback.

## O que este módulo não fará

- possuir dados de negócio, tenant, usuários ou segredos;
- abrir conexão, executar migrations ou importar drivers SQL;
- substituir workflow, documentos, lotes, aprovações ou efeitos externos;
- enviar SQL, HTML, CSS ou componentes por API;
- usar ORM.

## Contratos

Os contratos canônicos são governados pela Clear Platform:

- `clear.crud.definition.v1`;
- `clear.crud.datasource.v1`;
- `clear.crud.http.v1`;
- `clear.crud.renderer.v1`.

Este repositório contém a implementação e os testes de conformidade. O contrato
não é duplicado aqui: uma mudança semântica começa na Clear Platform, recebe
versão e só então é implementada neste módulo.

## Estado atual

- PRs 1 a 5: tipos públicos, definições lacradas, serviço com autorização e
  escopo, mutações transacionais auditáveis e suíte de conformidade pública.
- PR 6: `sqladapter.SimpleTable` executa a conformidade contra SQLite real,
  sem importar driver no código publicado.

O adapter SQLite é a referência inicial de `simple_table`. Postgres, MySQL e
MariaDB ainda não foram implementados; a API permanece desacoplada deles e os
adapters futuros executarão a mesma suíte de conformidade.

## Desenvolvimento

```bash
gofmt -w .
go test ./...
go test -race ./...
go vet ./...
```

Veja [AGENTS.md](AGENTS.md) e [guia de migração](docs/migrations.md) antes de
alterar a API pública.
