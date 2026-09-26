# Observabilidade do Clear CRUD

## Auditoria de mutações

Create, update e delete geram um AuditEvent obrigatório na mesma UnitOfWork da
mudança. O core não chama serviços remotos para isso.

Em produção, o AuditSink deve escrever um evento pequeno em tabela local ou
outbox transacional. A projeção para busca, SIEM, data lake ou alertas acontece
fora da transação, por worker com retry e retenção próprios.

Para recursos `simple_table`, `sqladapter.NewAuditSink` recebe um `*sql.DB` e
um mapping fechado da tabela de auditoria. Ele usa automaticamente a transação
aberta por `SimpleTable.Within`, desde que ambos recebam o mesmo banco. O
mapping inclui somente escopo, recurso, ação, ID, versões, principal, instante
e correlation ID; não possui campos para valores de formulário.

Falha para gravar a auditoria obrigatória aborta a mutação e retorna somente um
erro público temporário. A causa técnica não é apresentada ao operador.

Quando a política do host desliga deliberadamente a persistência de auditoria,
ele deve declarar `crud.NoopAuditSink{}` no bootstrap. O desligamento é uma
decisão operacional explícita, não um `nil` acidental, e não transforma logs de
acesso em prova de mutação.

## Telemetria de leitura e acesso

Listagens, consultas individuais e lookups não geram AuditEvent no v1.
Telemetria operacional desses acessos pertence ao host e pode ser ligada,
desligada, amostrada ou limitada por taxa. Ela é assíncrona e nunca altera o
resultado, a autorização ou a disponibilidade de um CRUD.
