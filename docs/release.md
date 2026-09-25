# Release dos pacotes frontend

Os pacotes `@clear-platform/crud-client` e `@clear-platform/crud-vue` são
artefatos experimentais versionados junto do core. Antes de uma publicação, a
mesma revisão deve passar:

1. `make validate` — inclui build, testes, cobertura, vulnerabilidades e
   `npm pack --dry-run` dos dois pacotes;
2. verificação de que os tarballs contêm somente artefatos públicos, sem testes,
   dados, segredos ou fonte do produto consumidor;
3. revisão do contrato `clear.crud.renderer.v1` e da compatibilidade SemVer;
4. definição explícita do registry, propriedade do escopo e licença aplicável;
5. publicação dos dois pacotes com a mesma versão e, somente depois, tag Git e
   release pública correspondente.

O repositório não fixa registry, token de publicação nem licença por suposição.
Essas decisões pertencem ao proprietário da distribuição. Até a publicação, um
produto consumidor não usa caminho local, `replace`, cópia, fork ou `vendor`.
