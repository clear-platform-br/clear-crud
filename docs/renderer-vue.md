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
`CrudScreen` é sempre conteúdo embutível: o produto hospedeiro fornece o shell,
viewport, header, menu lateral, rodapé e tema global. O renderer não oferece
modos standalone/embed nem altera o layout externo da aplicação.

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

### Título contextual do editor

O título fixo da definição continua sendo o fallback. Quando o host abre um
registro pai e precisa identificar a linha escolhida (por exemplo, uma tabela
auxiliar), a definição pode declarar um campo de título de contexto:

```go
Presentation: crud.Presentation{
    Collection:  crud.CollectionTable,
    Density:     crud.DensityComfortable,
    TitleField:  "title",
},
```

Em um registro existente, o renderer Vue usa o valor desse campo no cabeçalho
do modal. Na inclusão, ou quando o valor estiver vazio, usa o label fixo da
definição. O campo precisa ser visível e não sensível; a validação ocorre no
bootstrap. Isso altera apenas a apresentação: não muda escopo, autorização,
campos enviados ou persistência. O host não deve hardcodar o nome de um domínio
no renderer.

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
navegador. Árvore, filho de filho, muitos-para-muitos, workflow e efeitos
externos continuam fora do renderer. A release experimental atual ainda não
oferece anexos; quando a capability genérica for publicada, upload,
substituição, remoção e miniaturas serão habilitados somente por definição
explícita do recurso e por um repositório de armazenamento registrado no
servidor.

## Tema e extensão

Os componentes publicam `data-clear-crud-part` e `data-clear-crud-action`.
Importar `theme-default.css` é opcional. Produtos podem sobrescrever apenas as
custom properties `--clear-crud-*` ou fornecer tema integral; não devem mudar
a semântica do controller nem enviar estilos pela definição do backend.

O tema padrão oferece alternância de modo escuro e densidade compacta. Essas
preferências são locais ao navegador e à chave do recurso, portanto sobrevivem
ao refresh sem alterar a definição nem o backend. Na grade, textos ficam à
esquerda, números à direita e booleanos centralizados; os símbolos booleanos
podem ser declarados em `Field.BooleanDisplay`.

O toolbar também publica `Filtros` sem configuração adicional. O renderer gera
um controle para cada coluna pública de `Grid.Columns`; enums e booleanos usam
checkboxes e aceitam várias opções, enviando uma operação `in` com os valores
canônicos. `Todos` remove o filtro daquela coluna. Texto/lookup usam busca por
contém e tipos numéricos/data usam igualdade.

O `theme-default.css` é separado por assuntos em seções estáveis: tokens,
fundação/escopo do tema, controles compartilhados, toolbar, grade, paginação,
feedback (sucesso, aviso e erro), modal/formulário, lookup, enum,
acessibilidade e responsividade. O host continua responsável por Header, Menu,
Sidebar e Footer. Avisos usam o token âmbar, erros usam vermelho e sucesso usa
o verde da ação primária; isso permite trocar a paleta sem localizar regras
espalhadas pelos componentes.

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
