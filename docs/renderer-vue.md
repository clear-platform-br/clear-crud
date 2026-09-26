# Renderer padrão Vue

O renderer experimental é composto por dois pacotes versionáveis neste
repositório:

- `@clear-platform-br/crud-client`: transporte HTTP v1, estado, concorrência,
  paginação, mutation, erros públicos e mestre-detalhe;
- `@clear-platform-br/crud-vue`: componentes Vue 3, tema opcional e composição da
  tela.

O produto não escreve grade, formulário, paginação, busca, cliente HTTP,
handler ou feedback CRUD paralelo. Ele fornece um `CrudTransport` configurado
para o endpoint fechado do produto e monta `CrudScreen` pela chave do recurso.

```ts
import { HttpCrudClient } from '@clear-platform-br/crud-client'
import { CrudScreen, ptBR } from '@clear-platform-br/crud-vue'
import '@clear-platform-br/crud-vue/theme-default.css'

const client = new HttpCrudClient({ baseUrl: '/api/v1/crud' })
```

```vue
<CrudScreen resource="contacts" :client="client" :messages="ptBR" :resolve-message="translate" />
```

`resolveMessage` pertence ao produto e traduz rótulos e ajuda definidos pelo
servidor. O catálogo `ptBR` ou `enUS` cobre apenas a interface comum do motor;
o renderer não contém vocabulário de domínio.

Quando a definição autoriza `help`, o renderer apresenta o botão `?` e abre a
ajuda declarada. Ajuda de campo também usa `?`; edição usa lápis, inclusão usa
`+` e remoção usa `×`, sempre com rótulo acessível. A confirmação informa se
a ação arquiva ou elimina definitivamente o registro.

Campos `create_only` aceitam valor na inclusão e aparecem desabilitados durante
a edição. O client os omite da mutation de update; o backend continua sendo a
autoridade e também os rejeita se forem enviados diretamente.

## Mestre-detalhe

Quando a definição pública expõe `Details`, o editor apresenta as coleções
filhas diretas. O usuário inclui, edita ou remove itens e o client envia uma
única mutation. O campo interno de vínculo não é renderizado, nem aceito do
navegador. Árvore, filho de filho, muitos-para-muitos, upload, workflow e
efeitos externos continuam fora do renderer.

## Tema e extensão

Os componentes publicam `data-clear-crud-part` e `data-clear-crud-action`.
Importar `theme-default.css` é opcional. Produtos podem sobrescrever apenas as
custom properties `--clear-crud-*` ou fornecer tema integral; não devem mudar
a semântica do controller nem enviar estilos pela definição do backend.

## Galeria de temas planejada

O renderer terá exemplos visuais para demonstrar o esforço de customização:

1. tokens: cores, tipografia e espaçamento;
2. tema CSS completo usando os mesmos componentes;
3. composição por partes registradas;
4. renderer próprio usando o mesmo client e contrato.

Os exemplos devem passar as mesmas fixtures e cenários comportamentais. O
futuro `GoDataGrid` seguirá a mesma separação entre semântica estável e tema,
para que a aparência possa mudar sem duplicar persistência ou regras CRUD.

## Estado de entrega

Este renderer é `experimental`. O código está pronto para o piloto Vue, mas um
produto só o usa após a publicação e fixação da versão correspondente. React, React Native e Flutter
continuam planejados, sem promessa de implementação até haver consumidor real.
