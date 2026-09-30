# Manual de uso do `clear-crud` para agentes

Este manual define **quando usar** e **como consumir** a capability pública
`clear.crud`. É obrigatório antes de propor, iniciar ou alterar um CRUD em um
produto Clear.

Fonte dos contratos: `clear_platform` (`clear.crud.definition.v1`,
`clear.crud.datasource.v1`, `clear.crud.http.v1` e
`clear.crud.renderer.v1`). Este documento ensina o uso; não altera contratos.

## Mapa legível por máquina para agentes

Antes de escolher uma diretiva, o agente deve consultar
[`settings-catalog.json`](settings-catalog.json). O arquivo organiza cada
setting por grupo, default, exceção e critério de aplicação. Ele não é uma
segunda fonte de contrato: serve para evitar que uma capability já disponível
seja esquecida ou aplicada no lugar errado.

Para distinguir o que pode ser usado agora do que ainda é planejamento,
consulte também o [mapa atual de capabilities](capability-map.md). Uma entrada
no backlog não autoriza o agente a criar uma implementação local equivalente.

O motor não conhece o domínio dos dados. Uma tabela de estados, uma tabela da
FIPE, uma base de CEP, aeroportos ou voos são apenas recursos registrados pelo
host. O consumidor fornece migrations/importadores e declara as relações; o
engine executa a mesma mecânica de escopo, autorização, busca, paginação,
lookup e cache para todos eles.

Catálogos compartilhados devem usar `RegisterAutoGlobalTable` somente quando a
decisão for explícita: o recurso fica global, somente leitura e sem cópia por
tenant. Lookups dependentes declaram `WithLookup` no campo do formulário. O
cache read-through é server-side, limitado e separado por escopo confiável;
uma implantação com várias instâncias pode injetar um cache compartilhado.
O cache em memória padrão usa no máximo 512 entradas e TTL de cinco minutos;
o host pode fornecer outro `LookupCache` e deve limpá-lo após publicar uma
alteração de catálogo.
Se um tenant precisar restringir opções, mantenha o catálogo global e crie uma
associação tenant-owned — não replique o catálogo inteiro.

### Escopo confiável e política de manutenção

`tenant_id` é a fronteira padrão dos dados pertencentes ao tenant. Quando uma
visão server-owned precisa de outra partição obrigatória, o consumidor pode
declarar a chave adicional em `Definition.Scope.Keys` e mapeá-la no adapter por
`ScopeColumns`. O `ScopeProvider` fornece o valor confiável em cada operação;
o navegador nunca escolhe nem substitui essa chave. Isso é escopo de segurança,
não um filtro visual opcional.

Se a necessidade for somente filtrar a grade sob escolha do operador, use os
filtros de coluna publicados ou uma definição/recurso de leitura separado. Não
transforme um toggle do cliente em escopo e não use `LookupDefinition.FixedFilters`
para a grade: filtros fixos pertencem a lookups.

Em uma relação pai-filhos, a política de manutenção pertence ao registro pai;
ela não deve ser repetida nem reinterpretada em cada opção filha. Os valores
`system`, `customizable` e `user` são uma convenção possível do produto, não um
enum do motor: `system` identifica catálogo técnico fornecido pelo produto,
`customizable` permite manutenção operacional dos itens e `user` fica reservado
para produtos que permitem ao usuário criar catálogos. O clear-crud apenas
executa o escopo, autorização e definição declarados pelo consumidor.

Esse escopo adicional não deriva uma política do pai para uma rota independente
do filho: a coluna/alias precisa existir na fonte do próprio recurso. Hoje não
há um predicado fixo server-owned publicado que seja aplicado automaticamente
a `List`, `Get`, `Create`, `Update` e `Delete` de um filho a partir de um campo
do pai. Se essa garantia for necessária, pare no consumidor e abra uma evolução
genérica (ou um modo detail-only); não faça filtro local, endpoint paralelo ou
regra de TdT no produto.

## Tradução e Weblate

Use `MessageCode` estável nas definições e nos componentes publicados. O texto
traduzido fica em catálogos versionados no Git; não use o texto de um idioma
como chave e não faça o produto consultar o Weblate durante a execução.

O padrão operacional da família `clear*` é uma única instância self-hosted do
Weblate em Docker/Compose. Separe os produtos por projeto e, dentro de cada
projeto, separe componentes de frontend, backend e documentação. As permissões
devem ser dadas por projeto/componente; não misture no mesmo projeto produtos
sem vocabulário ou equipe em comum. Glossários e memória compartilhada só
devem ser habilitados para termos realmente comuns.

O Git continua sendo a fonte de verdade: o Weblate serve para edição,
revisão, validação e sincronização, preferencialmente por alterações revisáveis
no repositório. O runtime deve funcionar com os catálogos publicados mesmo que
o Weblate esteja indisponível.

Na instalação inicial, preserve os volumes do Weblate, PostgreSQL e Valkey e
automatize backup e restauração. Um servidor dedicado não é requisito inicial;
reavalie-o apenas por escala, isolamento contratual, compliance ou necessidade
de manutenção independente. Um produto consumidor configura apenas seu
resolver/catálogo e continua sem depender de código específico do Weblate no
core, adapter ou renderer.

Um lookup sem `Dependencies` é a raiz da cadeia: ao acionar o controle de
opções, o renderer carrega a primeira página sem exigir um ID ancestral. Um
campo filho fica bloqueado até que suas dependências tenham valor; depois, se
nenhum filho existir, o renderer informa que não há opções disponíveis. O banco
continua armazenando as chaves; a grade resolve e exibe os labels pelo mesmo
lookup registrado, nunca expõe o ID cru como texto de apresentação.
Para catálogos grandes, a busca digitada exige três caracteres por padrão e só
então gera uma requisição filtrada; a seta continua abrindo a primeira página
limitada. `LookupDefinition.MinSearchLength` é a exceção declarativa para
catálogos que precisam de outro limite.

As dependências filtram dinamicamente a tabela-alvo com valores já escolhidos
no formulário. Para particionar uma tabela auxiliar compartilhada, declare
`LookupDefinition.FixedFilters`, por exemplo
`Values: []crud.Value{"municipio"}`, no bootstrap do servidor. Um valor aplica
igualdade e vários aplicam pertencimento; eles se combinam com dependências,
escopo confiável e archive. A lista não é parâmetro HTTP, não aceita SQL livre
e não substitui tenant scope, autorização ou a política de archive.
Se o lookup deve mostrar um único catálogo, informe uma única ID (por exemplo,
`Values: []crud.Value{"30"}`). Informar `{"30", "40"}` é uma união deliberada
dos dois catálogos, não uma escolha entre eles; para escolhas independentes,
registre lookups/definições separados.

Exclusão é uma decisão separada do fato de a tabela existir: sem diretiva, o
CRUD não oferece delete; `WithSoftDelete("archived")` habilita archive; e
`WithHardDelete()` habilita `DELETE` físico. O último é uma exceção revisável,
mutuamente exclusiva com archive, e não pode ser inferido pelo agente a partir
do nome da tabela.

O comportamento default é `active_only`. Para uma grade que precise consultar
os dois estados, declare `Grid.ArchiveVisibility =
crud.ArchiveVisibilityActiveAndArchived` (ou a opção equivalente do adapter);
o renderer apresenta o toggle curto `Excluídos`, com tooltip e nome acessível
`Incluir registros excluídos`. O navegador só solicita a visibilidade já
declarada pelo servidor. Exibir arquivados não implica permitir edição ou
restauração: essas ações exigem capability própria. No grid, uma linha arquivada
é marcada explicitamente como `Excluído` na coluna de ações e recebe uma faixa
visual própria; `enabled=false` continua sendo apenas um estado de negócio.

## Princípio de evolução: defaults primeiro

O objetivo da capability é que o caso comum seja resolvido por convenção, com
uma entrada equivalente a:

```go
crud.Auto("contact_records")
```

O motor deve descobrir campos, tipos, chave, busca, ordenação, paginação,
lookups, grade, formulário e ações padrão a partir dos metadados do adapter.
O agente só acrescenta diretivas quando houver uma exceção: esconder uma
coluna, trocar rótulo, restringir campos graváveis, declarar validação de
domínio ou resolver uma relação ambígua.

No adapter convencional, `RegisterAutoTenantTable(ctx, db, registry, table)` é
o primeiro exemplo mínimo: `tenant_id` continua sendo um escopo confiável e,
sem uma política de archive explicitamente declarada, a definição não oferece
ação de exclusão. A ausência de `WithSoftDelete(...)` nunca faz o motor inferir
ou expor soft delete por nome de coluna.

Enums e `CHECK`s finitos seguem a mesma regra: quando o adapter os expõe no
bootstrap, os valores permitidos entram automaticamente no `FieldEnum`; a
definição do CRUD pode somente trocar labels/ordem ou restringir o subconjunto.
Sem metadado estrutural, o agente declara as opções explicitamente e não pode
inventar valores fora de uma restrição existente.

Defaults seguem a mesma fronteira: `Field.Default` ou a conveniência
`sqladapter.WithDefault` aceita somente um escalar estático. O renderer mostra
o valor no formulário de inclusão e o core aplica o mesmo valor quando uma
criação o omite; uma atualização nunca substitui o valor existente por default.
O adapter pode promover literais seguros do schema, mas expressões como
`CURRENT_TIMESTAMP` continuam sob responsabilidade do banco/adapter e não
viram dados ou regras no core.

Um default também pode pertencer a um campo técnico invisível (`Visible: false`)
quando o consumidor precisa preencher uma coluna interna sem expô-la ao
operador. Nesse caso o campo não aparece na definição pública nem pode ser
enviado pelo navegador; o core aplica o literal somente na criação. `Sensitive`
e `ReadOnly` continuam impedindo defaults estáticos.

Padrões declarativos usam `Field.Pattern` com sintaxe RE2 e, opcionalmente,
`Field.PatternMessage`; no `AutoTable`, a forma equivalente é
`sqladapter.WithPattern`. Eles são compilados no bootstrap e validados pelo
core em toda criação e atualização; o renderer recebe somente metadados
server-owned e o HTTP nunca aceita a expressão. Restrinja o padrão a valores
string-backed e use um `MessageCode` próprio quando a mensagem genérica não
for suficiente. Para os formatos recorrentes, prefira as constantes públicas
de `patterns.go`, que mantêm a definição legível sem esconder a possibilidade
de um RE2 customizado:

| Grupo | Constantes principais |
| --- | --- |
| Presença e espaços | `PatternNonBlank`, `PatternNoWhitespace`, `PatternNoLeadingOrTrailingWhitespace` |
| Letras e palavras | `PatternLettersOnly`, `PatternUppercaseLetters` (`PatternAllUpper`), `PatternLowercaseLetters` (`PatternAllLower`), `PatternFirstLetterUpper`, `PatternFirstLetterLower`, `PatternWords` |
| Letras, números e valores simples | `PatternLettersAndNumbers` (`PatternAlphaNumeric`), `PatternLettersNumbersAndSpaces` (`PatternAlphaNumericWithSpaces`), `PatternDigitsOnly`, `PatternUnsignedInteger`, `PatternSignedInteger`, `PatternDecimal` |
| Identificadores e códigos | `PatternUppercaseCode`, `PatternLowercaseCode`, `PatternUpperCamelCase`, `PatternLowerCamelCase`, `PatternSnakeCase`, `PatternKebabCase`, `PatternSlug`, `PatternDotSeparatedKey`, `PatternTwoLetterCode`, `PatternThreeLetterCode` |
| Formatos recorrentes | `PatternHexadecimal`, `PatternHexColor`, `PatternUUID`, `PatternVersion`, `PatternTime24Hour`, `PatternE164Phone`, `PatternHTTPURL`, `PatternMACAddress` |

As constantes de letras usam classes Unicode (`\\p{L}`, `\\p{Lu}` e
`\\p{Ll}`); códigos e identificadores ASCII deixam isso explícito no nome e
na expressão. `FieldEmail`, `FieldPhone`, `FieldDate`, `FieldDateTime` e
`FieldDecimal` continuam sendo preferíveis quando o tipo normalizado já
expressa a semântica do campo. Os presets validam formato, não normalizam
entrada nem substituem regras de domínio.

Exemplo:

```go
sqladapter.WithPattern(
    "name",
    crud.PatternFirstLetterUpper,
    "crud.validation.name_uppercase",
)
```

### Modelos de leitura registrados

Quando a grade precisa mostrar um valor vindo de outra tabela ou calculado,
registre uma definição server-owned com um `DataSource` de leitura do adapter.
No adapter SQL desta release, use `sqladapter.RegisterReadModel` e mantenha a
consulta no bootstrap do servidor; ela é validada uma vez e não é recebida do
navegador.
Quando os aliases SQL coincidirem com as chaves públicas, o registro infere
`Fields`, `ScopeColumns`, `Grid.Searchable` e `Grid.Sortable` a partir da
definição. Informe esses mapas apenas quando houver um alias físico diferente
ou uma allowlist mais restrita.
Declare o campo retornado pelo modelo com `ReadOnly: true`, inclua-o somente em
`Grid.Columns` quando fizer sentido e deixe-o fora de `Form.Fields`. O adapter
executa o join ou cálculo e pode marcar esse metadado no bootstrap por meio de
`SourceMetadata`; o core então expõe o valor para leitura e rejeita qualquer
tentativa de gravá-lo.

### Padrão dos exemplos de definição

Os exemplos do demo seguem deliberadamente a mesma sequência de comentários,
para que um CRUD novo possa ser construído por repetição do padrão:

1. tabela física e chave pública do recurso;
2. opções, validações, defaults e política de exclusão;
3. lookups, dependências ou relações, quando existirem;
4. projeção e ordem do grid;
5. projeção e ordem do formulário;
6. registro final no adapter/registry.

Os comentários explicam a intenção de cada bloco; não substituem a definição
nem introduzem regra especial no demo, no core ou no renderer.

O navegador nunca envia SQL, tabela, coluna ou regra de join. O core apenas
valida a allowlist de campos, filtros e ordenação já registrada; a tradução da
consulta pertence ao adapter de cada banco. Um modelo de leitura não é um
atalho para alterar o schema nem para transformar campos calculados em dados
mutáveis.

Outros adapters devem publicar uma capability equivalente antes de um produto
depender dela. Não simule o recurso no produto com um CRUD local nem mova a
consulta para o core.

Modelos de leitura também não são uma capability de relatório: eles retornam
linhas e podem trazer joins ou cálculos escalares. Agrupamentos, somas,
subtotais e total geral exigem uma capability agregada própria, com medidas e
grupos declarados pelo servidor e execução no adapter. Nunca aceite `GROUP BY`,
expressões ou SQL enviados pelo navegador.

A configuração automática nunca decide silenciosamente segurança. O host deve
resolver, por perfil, os gates `open`, `read`, `write`, `soft_delete` e
`hard_delete`. O renderer omite ações não autorizadas, mas o backend sempre
revalida cada gate.

Para o caso convencional, a consulta da grade e a consulta do formulário são
distintas: a primeira retorna só colunas úteis para localizar o registro; a
segunda roda após a seleção e retorna os campos permitidos para edição. SQL
customizado, quando necessário, permanece fechado no servidor.

## Regra de 30 segundos — use esta primeiro

Responda às três perguntas, nesta ordem:

1. O operador só administra registros individuais: listar, buscar, incluir,
   editar, arquivar/excluir?
2. Salvar esse registro não dispara envio, pagamento, documento, aprovação,
   lote, integração externa ou etapa humana especial?
3. A release pública escolhida já possui **adapter de dados e renderer padrão**
   para este caso?

| Respostas | Decisão obrigatória | Próxima ação |
| --- | --- | --- |
| 1 = sim, 2 = sim, 3 = sim | `USE CLEAR-CRUD` | Siga a receita de cinco passos abaixo. |
| 1 = não ou 2 = não | `CASO PRÓPRIO` | Escreva a tela/caso de uso do produto; não recrie motor CRUD. |
| 3 = não | `BLOQUEADO POR LACUNA` | Não escreva fallback local. Registre a lacuna no `clear-crud`. |

Não existe quarta saída. Em especial, “vou criar uma pequena tabela/paginação
temporária no produto” é proibido.

## Receita de cinco passos — quando a decisão for `USE CLEAR-CRUD`

1. **Fixe a release pública.** Sem `replace`, cópia, fork, `vendor` ou
   `internal`.
2. **Use o bootstrap oficial já existente no host.** Não crie outro por recurso.
3. **Crie uma definição curta.** Somente chave, campos, rótulos, permissões,
   escopo, lookups, validações e política de exclusão.
4. **Monte o renderer oficial pela chave do recurso.** Não escreva tabela,
   formulário, paginação, busca, cliente HTTP, handler ou mensagens CRUD.
5. **Teste a definição e a integração.** Escopo, permissão, validação,
   concorrência, exclusão e ausência de PII onde aplicável.

Além de migration, tradução/tema e testes, a integração de um recurso não pode
criar código genérico. Se parecer que são necessários dois ou mais arquivos de
infraestrutura CRUD no produto, pare: a decisão está errada ou a release pública
não cobre o caso.

## Regra de propriedade

`clear-crud` possui comportamento genérico: definição, validação estrutural,
escopo, autorização, busca, paginação, concorrência, exclusão, auditoria,
transporte e renderer padrão.

O produto consumidor possui somente dados e significado: migration, definição
do recurso, campos, rótulos, permissões, lookups registrados e validações de
domínio.

Um produto **não pode** implementar localmente grade, paginação, formulário,
cliente HTTP, handler, repository ou adapter CRUD genérico. Se uma capability
necessária não existe na release pública, isso é uma lacuna do `clear-crud`, não
autorização para criar fallback local.

## Critério objetivo: quando usar

Um caso **deve usar `clear-crud`** se todas as respostas forem “sim”:

1. O operador administra registros persistidos de um recurso definido.
2. O fluxo principal é listar/buscar, paginar, incluir, consultar, editar e
   arquivar ou excluir registros individuais.
3. Os campos são escalares ou lookups já suportados pelo contrato publicado.
4. Escopo, permissão, validação, concorrência e política de exclusão podem ser
   declarados para o recurso.
5. A release pública selecionada oferece adapter de dados e renderer padrão
   necessários, sem código genérico adicional no produto.
6. A operação não exige documentos, etapas, confirmação humana especial,
   efeito externo, processamento em lote ou coordenação entre recursos como
   parte da mesma ação — exceto o mestre-detalhe simples de um nível já
   suportado pela release pública.

Um caso **deve ser escrito como caso de uso/tela própria** se qualquer resposta
abaixo for “sim”:

1. A ação executa ou confirma efeito externo: envio, pagamento, emissão,
   assinatura, captura ou integração com provedor.
2. O formulário é mestre-detalhe com árvore, múltiplos níveis, relação muitos
   para muitos, submissão parcial implícita, multietapa ou tem invariantes
   transacionais que não cabem nos campos e hooks públicos publicados. Anexos
   simples só entram no `clear-crud` quando a release consumida publicar a
   capability de anexos opt-in; documentos com fluxo, aprovação ou efeitos
   externos continuam sendo caso próprio.
3. O operador toma decisão de negócio que exige revisão explícita, prova,
   documentos ou aprovação humana.
4. A experiência principal é operação, painel analítico, linha do tempo,
   fluxo de trabalho ou lote; não administração de registros individuais.
5. A release pública não oferece a capability necessária.

No segundo grupo, a tela própria pode reutilizar **somente** componentes e
tokens visuais oficiais já publicados. Ela não recria o motor CRUD.

## Filtro obrigatório antes de escrever código

Antes de qualquer arquivo de implementação, registre esta decisão no PR, issue
ou documento de decisão do produto:

```md
## Decisão de CRUD: <recurso>

- Classificação: `clear-crud` | `caso de uso próprio`
- Recurso persistido: <nome lógico; nunca SQL vindo do navegador>
- Critérios 1–6: sim/não, com justificativa de uma frase quando for não
- Release pública: `<módulo>@<versão>`
- Adapter e renderer públicos: <nomes e versões>
- Escopo confiável: <ex.: tenant_id vindo da sessão>
- Permissões: <ler/criar/editar/apagar>
- Política de exclusão: `none` | `archive` | `hard_delete`
- Regra de domínio fora do CRUD, se houver: <uma frase>
```

### Exclusão lógica

`archive` é exclusão lógica: o adapter deve excluir esses registros de toda
lista, leitura e mutação normal. Um produto só pode expor registros arquivados
em uma definição de consulta/auditoria explicitamente declarada; o navegador
nunca habilita essa visibilidade por conta própria. Estados como ativo,
cancelado ou encerrado não são exclusão lógica e nunca são inferidos pelo nome
de uma coluna.

Decisão `clear-crud` sem release, adapter e renderer públicos identificados é
inválida. A ação correta é parar e abrir uma lacuna no repositório proprietário;
nunca construir equivalentes no produto.

Para Vue, o repositório contém os pacotes experimentais
`@clear-platform-br/crud-client` e `@clear-platform-br/crud-vue`. O produto importa
`CrudScreen` somente após a publicação da release, passa a chave do recurso, client HTTP fechado e tradução,
e pode importar o tema padrão ou sobrescrever seus tokens. Veja
[`renderer-vue.md`](renderer-vue.md). Não implemente variações locais do
controller, da tabela ou do formulário.

## Como usar `clear-crud`

### 1. Escolha e fixe uma release pública

O produto declara versão publicada no seu gerenciador de dependências. É
proibido usar cópia, fork, `vendor`, `replace` local ou pacote `internal`.

### 2. Use o bootstrap oficial do host

O host configura uma única vez ports confiáveis de identidade, escopo,
autorização, auditoria, tradução e relógio. O navegador nunca fornece tenant,
tabela, coluna, SQL ou nome de adapter. Essa infraestrutura não é repetida por
recurso.

### 3. Declare o recurso

Cada cadastro fornece apenas uma definição revisável:

- chave lógica fechada;
- títulos e ajuda;
- escopo obrigatório;
- permissões por ação;
- campos, tipos, obrigatoriedade, limites e sensibilidade;
- colunas de lista, busca, ordenação e paginação;
- hints semânticos de apresentação, como densidade e símbolos explícitos para
  campos booleanos;
- lookups registrados;
- política de exclusão;
- validações de domínio que o contrato comporta;
- adapter público configurado por allowlist.

A definição não contém CSS, HTML, componente Vue/React/Flutter, SQL recebido do
navegador, segredo, tenant livre ou regra de fluxo externo.

O caso automático mínimo fica concentrado no bootstrap do host. Um catálogo
compartilhado e um CRUD tenant-owned podem ser registrados assim:

```go
if err := sqladapter.RegisterAutoGlobalTable(ctx, db, registry, "ref_states"); err != nil {
    return err
}
if err := sqladapter.RegisterAutoTenantTable(ctx, db, registry, "contacts",
    sqladapter.WithLookup("state_id", crud.LookupDefinition{
        Resource: "ref_states", ValueField: "id", LabelField: "name", PageSize: 25,
    }),
); err != nil {
    return err
}
```

O nome da tabela continua sendo configuração do servidor. O navegador recebe
somente a definição pública e as rotas fechadas do adapter HTTP; ele nunca
escolhe o recurso-alvo do lookup.

`Grid.Columns` é a allowlist da grade e não precisa conter todos os campos do
recurso. A geração automática limita a grade a oito colunas por padrão; uma
exceção explícita, como `WithGridColumns("name", "status")`, escolhe outra
projeção sem retirar os demais campos do editor.

A grade oferece duas camadas sem configuração extra. O campo `Buscar registros`
envia `q` para uma busca textual global; o painel `Filtros` deriva um controle
de cada coluna pública da grade e envia `Query.Filters` tipados. Booleanos usam
`Todos`, `Sim` e `Não`; enums usam `Todos` e as opções declaradas. O adapter
recebe sempre o valor canônico persistido (`review`), nunca o label traduzido
(`Em análise`). Texto e lookup usam `contains`; números, datas e datetime usam
igualdade; a validação final da allowlist e do tipo continua no servidor.
Quando mais de uma opção enum/boolean é marcada, o renderer envia uma única
operação `in` com os valores canônicos: o resultado é “qualquer opção marcada”.

O tamanho inicial da página também é uma escolha do consumidor. A geração
automática usa 25 por padrão; `WithGridPageSize(10)` é uma exceção válida para
uma tela compacta ou uma demonstração. O serviço só aceita tamanhos declarados
na allowlist server-owned e nunca ultrapassa 100 registros por página.

No Lazy Mofo, `grid_sql` e `form_sql` acumulam três decisões diferentes: quais
campos a consulta retorna, em que ordem as colunas aparecem e, quando há
`ORDER BY`, em que ordem os registros são listados. No contrato do
`clear-crud`, a projeção e a ordem da grade são declaradas em `Grid.Columns`,
e a ordem dos campos do formulário é declarada em `Form.Fields`, com fallback
para a ordem server-owned de `Definition.Fields`. `Grid.DefaultSort` trata a
ordenação dos registros e continua sujeita à
validação de índices pelo adapter. Isso preserva a intenção útil sem aceitar
SQL livre na definição ou no navegador.

Joins, campos calculados e filtros que não possam ser expressos por campos,
lookups e filtros allowlisted não devem ser colocados em `Grid.Columns`. Eles
exigem uma projeção/read model registrada pelo servidor ou um caso de uso
próprio; nunca uma query recebida do cliente.

Na geração automática, a ordenação usa o primeiro índice compatível com campos
de negócio; prefixos técnicos de escopo podem antecedê-los no índice. Um índice
com prefixo de arquivamento não é considerado suficiente, porque o filtro
convencional de registros ativos aceita `NULL` ou `0` e não garante essa ordem.
Sem índice de negócio, a primeira coluna da lista é o fallback. Quando
a primeira coluna ordenada é booleana, a ordem padrão também usa a coluna de
negócio seguinte para manter os nomes determinísticos dentro de cada grupo.
Uma ordenação explícita, declarada com `WithDefaultSort(...)`, exige um índice
compatível e faz o bootstrap falhar se a tabela não o possuir. O adapter apenas
inspeciona o schema no bootstrap; ele não cria índices nem recebe ordenação do
navegador.

Para booleanos, o renderer padrão usa símbolos acessíveis e neutros. Se o
produto precisar de outra convenção, a definição pode declarar
`BooleanDisplay` no próprio campo, por exemplo `True: "●", False: "○"`. Isso é
configuração de apresentação do consumidor; não é inferida do banco nem recebe
HTML.

Use `create_only` para uma chave de negócio informada ao incluir e nunca
alterada depois (por exemplo, número de apartamento). O renderer a mantém
visível e desabilitada na edição; o backend a recusa em `update`. A identidade
técnica do registro continua opaca e é configurada somente no adapter
registrado pelo servidor.

### Mestre-detalhe simples, quando a release o oferecer

Use `Details` somente para um registro pai e coleções filhas diretas. Cada
filho continua sendo um recurso registrado, com seus próprios campos,
permissões e adapter. A definição do pai informa a coleção, o recurso filho,
o campo interno de vínculo e os limites mínimo/máximo. O campo de vínculo é
invisível e somente leitura: o motor o preenche; a tela nunca o envia.

Na mutation, a coleção informa explicitamente cada inclusão, edição versionada
ou remoção. Pai, filhos e auditoria usam a mesma `UnitOfWork`. Não use esta
capability para árvore, muitos-para-muitos, filho de filho, documentos com
workflow ou integração externa. Upload de anexo simples será uma capability
genérica separada e opt-in; ela ainda não faz parte da release experimental
atual.

### Tabela de Tabelas (TdT) com campos fixos

Uma tabela pai que descreve os campos dos próprios itens continua sendo um
mestre-detalhe genérico. O consumidor declara quais campos já existentes no
filho recebem metadados do pai; o core não conhece catálogo, tabela ou TdT:

```go
FieldMetadata: []crud.DetailFieldMetadataSource{
    {Field: "value_1", LabelField: "value_1_label", TypeField: "value_1_type", RequiredField: "value_1_required"},
    {Field: "value_2", LabelField: "value_2_label", TypeField: "value_2_type", RequiredField: "value_2_required"},
},
```

Os slots continuam fixos (`value_1` até `value_4` quando existirem no schema);
a capability não cria colunas nem altera migrations. O pai fornece o label
literal, o tipo permitido e a obrigatoriedade. Um slot mapeado cujo label
esteja vazio fica oculto e não entra na mutation; assim um catálogo de uma
coluna mostra somente, por exemplo, `Nome`. O primeiro slot continua sendo a
única exigência do contrato do consumidor, se essa for a regra do produto.

`DisplayLabel` é texto de apresentação, não chave de tradução. O renderer usa
o texto resolvido no campo filho; labels fixos continuam usando `MessageCode`.
O servidor repete a resolução para validar tipo, requiredness e campos aceitos,
portanto o navegador não é a autoridade. O título contextual do pai usa
`Presentation.TitleField` quando declarado.

Ao testar uma definição recebida por HTTP, valide o caminho completo
JSON → definição normalizada → `resolveDetailFields`. Não basta testar somente
um objeto TypeScript já tipado: o transporte deve preservar `FieldMetadata`
em PascalCase e camelCase antes de o renderer aplicar labels, tipos,
obrigatoriedade e visibilidade.

Essa convenção também atende outros mestres-detalhes com slots configuráveis;
os nomes da TdT aparecem somente na definição do consumidor e no demo.

### 4. Monte a UI oficial

A rota/tela importa o renderer oficial da mesma release. Ela entrega somente
contexto visual do produto e chave do recurso; o renderer consulta a definição
e usa o transporte oficial. O produto não implementa listagem, formulário,
paginação, busca, mutação ou mensagens CRUD em paralelo.

Se o editor precisar identificar o registro pai selecionado, declare o campo
`Presentation.TitleField` na definição. O renderer Vue usa seu valor no modal
de edição e cai no label fixo na inclusão ou quando o valor está vazio. O campo
precisa ser visível e não sensível. Essa é uma extensão visual server-owned:
não altera escopo, permissões, mutation ou dados persistidos. Não crie títulos
hardcoded no frontend nem use o nome de uma linha como regra do motor.

### 5. Valide a integração

O mínimo obrigatório é:

1. teste de registro da definição;
2. teste de isolamento por escopo confiável;
3. teste de autorização por ação;
4. teste de validação e concorrência quando houver mutação;
5. teste de política de exclusão;
6. teste de interface usando renderer oficial, sem PII em colunas não autorizadas;
7. cenário Gherkin do resultado operacional.

## Sinais de drift e ação obrigatória

| Sinal encontrado | Ação obrigatória |
| --- | --- |
| Produto cria `CrudTable`, `CrudForm`, `useCrud`, paginação, busca ou cliente CRUD próprio | Parar. Remover duplicação e consumir renderer/client oficial. |
| Produto cria handler CRUD genérico ou recebe tabela/SQL por HTTP | Parar. Usar transporte fechado do `clear-crud`. |
| Adapter ou renderer público não cobre a necessidade | Não criar bridge local. Abrir lacuna no `clear-crud` e aguardar release pública. |
| Recurso possui lote, documentos, aprovação ou efeito externo | Classificar como caso de uso próprio; reutilizar apenas primitives oficiais compatíveis. |
| Definição tenta esconder regra complexa em callback livre | Tirar regra para caso de uso próprio ou evoluir contrato público com evidência. |
| CPF, e-mail, telefone ou outro dado sensível aparece na grade por padrão | Corrigir definição/renderer e testar a ausência antes de liberar. |

## Regra de encerramento

Um CRUD não está concluído por “ter uma tabela na tela”. Só está concluído quando
a decisão existe, a definição é registrada pela release pública, o renderer
oficial é usado sem implementação concorrente e testes comprovam escopo,
autorização, validação, concorrência e exclusão aplicáveis.
