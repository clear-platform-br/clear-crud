# `@clear-platform/crud-client`

Client TypeScript experimental do contrato `clear.crud.http.v1`. Ele concentra
transporte HTTP, estado, concorrência, validação estrutural e mutation
mestre-detalhe; produtos não recriam esse comportamento.

Use somente uma versão pública fixada e um endpoint CRUD fechado pelo servidor:

```ts
import { HttpCrudClient } from '@clear-platform/crud-client'

const client = new HttpCrudClient({ baseUrl: '/api/v1/crud' })
```

Consulte o manual completo no repositório `clear-crud/docs/renderer-vue.md`.
