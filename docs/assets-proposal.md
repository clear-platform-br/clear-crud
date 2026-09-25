# Proposta: referências de arquivos em CRUDs

Esta é uma proposta para contrato futuro da Clear Platform; não altera
`clear.crud.definition.v1` nem introduz upload no core.

Um campo que represente imagem ou arquivo guarda somente uma referência opaca
emitida por serviço de assets autorizado. O navegador nunca escolhe caminho de
servidor, bucket, chave física ou tenant. Os bytes permanecem em storage
privado; banco e CRUD persistem apenas a referência validada.

O serviço de assets será responsável por autorização, escopo confiável,
allowlist de MIME, limite de tamanho, verificação de conteúdo, retenção e URL
temporária de leitura. A associação da referência ao registro é revalidada na
mutação. Nenhuma URL permanente, path local ou blob é exposto pela definição.
