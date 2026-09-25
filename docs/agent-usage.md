# Manual de uso do `clear-crud` para agentes

Este manual define **quando usar** e **como consumir** a capability pública
`clear.crud`. É obrigatório antes de propor, iniciar ou alterar um CRUD em um
produto Clear.

Fonte dos contratos: `clear_platform` (`clear.crud.definition.v1`,
`clear.crud.datasource.v1`, `clear.crud.http.v1` e
`clear.crud.renderer.v1`). Este documento ensina o uso; não altera contratos.

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
   transacionais que não cabem nos campos e hooks públicos publicados.
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

Decisão `clear-crud` sem release, adapter e renderer públicos identificados é
inválida. A ação correta é parar e abrir uma lacuna no repositório proprietário;
nunca construir equivalentes no produto.

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
- lookups registrados;
- política de exclusão;
- validações de domínio que o contrato comporta;
- adapter público configurado por allowlist.

A definição não contém CSS, HTML, componente Vue/React/Flutter, SQL recebido do
navegador, segredo, tenant livre ou regra de fluxo externo.

### Mestre-detalhe simples, quando a release o oferecer

Use `Details` somente para um registro pai e coleções filhas diretas. Cada
filho continua sendo um recurso registrado, com seus próprios campos,
permissões e adapter. A definição do pai informa a coleção, o recurso filho,
o campo interno de vínculo e os limites mínimo/máximo. O campo de vínculo é
invisível e somente leitura: o motor o preenche; a tela nunca o envia.

Na mutation, a coleção informa explicitamente cada inclusão, edição versionada
ou remoção. Pai, filhos e auditoria usam a mesma `UnitOfWork`. Não use esta
capability para árvore, muitos-para-muitos, filho de filho, upload, documentos,
workflow ou integração externa.

### 4. Monte a UI oficial

A rota/tela importa o renderer oficial da mesma release. Ela entrega somente
contexto visual do produto e chave do recurso; o renderer consulta a definição
e usa o transporte oficial. O produto não implementa listagem, formulário,
paginação, busca, mutação ou mensagens CRUD em paralelo.

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
