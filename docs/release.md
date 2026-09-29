# Release dos pacotes frontend

Os pacotes `@clear-platform-br/crud-client` e `@clear-platform-br/crud-vue` são
artefatos experimentais versionados junto do core. Antes de uma publicação, a
mesma revisão deve passar:

1. `make validate` — inclui build, testes, cobertura, vulnerabilidades e
   `npm pack --dry-run` dos dois pacotes;
2. verificação de que os tarballs contêm somente artefatos públicos, sem testes,
   dados, segredos ou fonte do produto consumidor;
3. revisão do contrato `clear.crud.renderer.v1` e da compatibilidade SemVer;
4. publicação privada no GitHub Packages em `https://npm.pkg.github.com`, sob a
   organização `clear-platform-br`;
5. publicação dos dois pacotes com a mesma versão e, somente depois, tag Git e
   release GitHub correspondente.

Versões pré-release, como `0.2.0-experimental.0`, são publicadas na tag npm
`experimental` e como pré-release no GitHub; elas nunca substituem a tag
`latest` de uma versão estável.

O workflow usa `GITHUB_TOKEN`; nenhum token é gravado no repositório. Até a
publicação, um produto consumidor não usa caminho local, `replace`, cópia, fork
ou `vendor`.

## Regra de compatibilidade

Depois que houver consumidores externos, uma release compatível só pode:

- adicionar entradas opcionais com o default antigo;
- adicionar campos opcionais a outputs, sem remover, renomear ou mudar o tipo e
  o significado dos campos existentes;
- corrigir falhas sem alterar o contrato observável de uma operação válida.

Toda mudança incompatível exige nova major, contrato versionado, guia de
migração e período explícito de coexistência. Testes de regressão devem provar
que uma definição sem a nova opção continua produzindo o comportamento anterior.
