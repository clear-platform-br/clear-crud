# Compatibilidade e migrations

Não há migration pertencente a este módulo. A implementação pública pode ser
versionada, mas migrations de tabelas, coleções, versão de registro e
arquivamento pertencem ao produto consumidor.

Migrations de tabelas, coleções, versão de registro e arquivamento pertencem ao
produto consumidor. O scaffold futuro poderá gerar sugestões revisáveis, mas
nunca executará migration em produção.

Uma mudança incompatível em tipos, contrato HTTP, definição pública ou semântica
de erro exige nova major, guia de migração específico e período explícito de
coexistência.
