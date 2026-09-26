# `@clear-platform-br/crud-vue`

Renderer Vue 3 experimental para `clear.crud`. Requer Vue 3.5 e
`@clear-platform-br/crud-client` na mesma versão.

```ts
import { HttpCrudClient } from '@clear-platform-br/crud-client'
import { CrudScreen, ptBR } from '@clear-platform-br/crud-vue'
import '@clear-platform-br/crud-vue/theme-default.css'
```

Monte `CrudScreen` com a chave lógica do recurso e um client configurado pelo
host. Não implemente tabela, formulário, paginação, busca ou controller CRUD
paralelos no produto.
