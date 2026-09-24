# Observabilidade do Clear CRUD

## Auditoria de mutações

Create, update e delete geram um AuditEvent obrigatório na mesma UnitOfWork da
mudança. O core não chama serviços remotos para isso.

Em produção, o AuditSink deve escrever um evento pequeno em tabela local ou
outbox transacional. A projeção para busca, SIEM, data lake ou alertas acontece
fora da transação, por worker com retry e retenção próprios.

Falha para gravar a auditoria obrigatória aborta a mutação e retorna somente um
erro público temporário. A causa técnica não é apresentada ao operador.

## Telemetria de leitura e acesso

Listagens, consultas individuais e lookups não geram AuditEvent no v1.
Telemetria operacional desses acessos pertence ao host e pode ser ligada,
desligada, amostrada ou limitada por taxa. Ela é assíncrona e nunca altera o
resultado, a autorização ou a disponibilidade de um CRUD.
