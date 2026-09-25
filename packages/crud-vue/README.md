# `@clear-platform/crud-vue`

Renderer Vue 3 experimental para `clear.crud`. Requer Vue 3.5 e
`@clear-platform/crud-client` na mesma versão.

```ts
import { HttpCrudClient } from '@clear-platform/crud-client'
import { CrudScreen, ptBR } from '@clear-platform/crud-vue'
import '@clear-platform/crud-vue/theme-default.css'
```

Monte `CrudScreen` com a chave lógica do recurso e um client configurado pelo
host. Não implemente tabela, formulário, paginação, busca ou controller CRUD
paralelos no produto.
