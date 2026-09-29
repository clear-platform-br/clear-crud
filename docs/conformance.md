# Conformidade de adapters

`conformance.TestDataSource` é a suíte pública que confirma o comportamento
de um adapter `DataSource` contra o contrato `clear.crud.datasource.v1`.
Ela não escolhe driver, banco, DSN, schema ou tenant: o repositório do adapter
fornece estes detalhes por meio de um `Fixture` isolado para cada subteste.

## Uso em um adapter real

```go
func TestPostgresDataSource(t *testing.T) {
	conformance.TestDataSource(t, newPostgresFixture)
}
```

O fixture deve criar uma base de teste vazia e descartável, duas `Scope`s
confiáveis diferentes, uma mutação válida e distinta para cada rótulo e a
consulta base que representa a ordenação permitida pelo recurso. Quando o
adapter declara `lookup`, ele também deve fornecer uma `LookupQuery` válida.
Quando declara `unit_of_work`, deve fornecer o `UnitOfWork` correspondente.

Não é válido atender essa suíte com mock para atestar persistência. SQLite,
PostgreSQL, MySQL e MariaDB devem executá-la contra o respectivo adapter e
banco reais em CI.

## Garantias verificadas

- contexto cancelado é respeitado;
- um registro nunca atravessa o limite de `Scope`;
- paginação, total e cursor obedecem às capabilities declaradas;
- atualização concorrente com a mesma versão produz exatamente um sucesso e
  um conflito;
- `none`, `archive` e `hard_delete` seguem a política declarada;
- após `archive`, o registro não aparece em `List` ou `Get` normais e não
  aceita `Update` normal;
- quando a definição declara `active_and_archived`, `List` aceita a consulta
  opt-in e marca os registros arquivados como somente leitura;
- lookup retorna somente opções públicas completas;
- uma mutação dentro de `UnitOfWork` é desfeita quando a transação falha.

Uma capability não declarada não é presumida. Uma capability declarada sem o
comportamento correspondente reprova o adapter. O core define esse
comportamento; o adapter apenas o traduz para sua persistência, sem criar
políticas próprias por banco ou por produto.

## Metadados estruturais no bootstrap

Um adapter pode implementar `crud.MetadataSource` para devolver fatos seguros
do schema durante `Registry.Register`. Quando uma coluna possui um conjunto
finito reconhecido por enum nativo ou `CHECK`, o core promove o campo para
`FieldEnum` e apresenta os valores por padrão. O adapter não é consultado em
requisições HTTP.

Uma definição explícita pode restringir esse conjunto a um subconjunto e trocar
labels ou ordem. Um valor fora do conjunto estrutural faz o registro falhar;
uma definição nunca amplia o que o banco aceita. O adapter SQLite reconhece
listas `IN (...)` e igualdades finitas ligadas por `OR`; outros adapters devem
normalizar seus enums nativos ou checks na mesma porta.
