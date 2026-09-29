# Demo descartável do Clear CRUD

Este diretório é apenas a primeira renderização local do renderer oficial. As
tabelas e o banco usam nomes com `clear_crud_` para deixar claro que não são
dados de produto.

Em dois terminais, a partir da raiz do repositório:

```bash
go run ./cmd/clear-crud-demo
npm run dev --workspace clear-crud-disposable-demo-web
```

Abra `http://127.0.0.1:5173`. O Vite encaminha `/api` para o host Go local.
A primeira grade usa a definição mínima, passando somente o nome de uma tabela
convencional; campos, tipos, busca, ordenação, paginação e formulário são
gerados pelos defaults. A segunda reutiliza a mesma tabela, mas limita a página
a 10 registros e traduz `status` para pt-BR. A terceira usa a tabela física
completa de 21 colunas para demonstrar enum, archive e projeções declaradas.
A quarta grade usa uma tabela pequena com quatro enums para demonstrar os
controles `select`, `radio`, `segmented` e `buttons`; em viewport móvel o
renderer usa `select` nativo para os quatro.
A quinta grade demonstra lookups encadeados, e a sexta confirma que um filtro
fixo declarado pelo consumidor particiona um catálogo global sem regra no
renderer: ao buscar `Real` no campo Estado, a moeda não aparece.
Para restaurar os 100 registros de cada tabela,
o índice de listagem e o schema completo, execute:

```bash
sqlite3 demo/clear_crud_disposable_junk.sqlite < demo/clear_crud_disposable_junk.sql
```

Como o `AutoTable` inspeciona o schema no bootstrap, reinicie manualmente o
processo Go depois de restaurar ou alterar a tabela para carregar a definição
atualizada; o frontend Vite pode permanecer em execução.
