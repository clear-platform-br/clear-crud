# language: pt
Funcionalidade: fundação pública do Clear CRUD
  Para que adapters e consumidores tenham um vocabulário estável
  Como implementador da capability clear.crud
  Quero tipos e erros públicos pequenos e sanitizáveis

  Cenário: erro público não vaza a causa interna
    Dado um erro interno de driver ou persistência
    Quando o core o associa a um Error com código publicável
    Então Error retorna apenas o código público em sua mensagem
    E a causa permanece disponível somente por errors.Unwrap
    E a serialização JSON não contém a causa

  Esquema do Cenário: ação pública usa identificador estável
    Quando um consumidor usa a ação <acao>
    Então o valor público é <valor>

    Exemplos:
      | acao   | valor  |
      | criar  | create |
      | ler    | read   |
      | editar | update |
      | apagar | delete |
      | ajuda  | help   |

  Cenário: registry lacrado não aceita recurso novo
    Dado um registry com uma definição válida
    Quando o host finaliza o startup e lacra o registry
    Então nenhuma nova definição pode ser registrada
    E a definição existente continua disponível para leitura

  Cenário: registry não deixa definição mutar depois do startup
    Dado uma definição válida com slices de campos e paginação
    Quando o host a registra e altera o valor original
    Então o registry preserva a cópia validada
    E o consumidor que lê a definição também recebe uma cópia defensiva

  Cenário: recurso mutável exige garantias reais
    Dado uma definição com criação, edição ou exclusão
    Quando o DataSource não declara atomic_version e unit_of_work
    Então o registro falha antes de o recurso ser exposto

  Cenário: alteração preserva o gate de regressão
    Dado uma alteração no core público
    Quando a suíte de validação é executada
    Então build, testes, race detector e vet passam
    E a cobertura global permanece no mínimo em 90 por cento
    E os ramos críticos alterados têm testes explícitos de sucesso e recusa

  Cenário: leitura usa escopo confiável antes do datasource
    Dado um recurso com escopo tenant_id obrigatório
    Quando o host não resolve o tenant_id do principal
    Então a operação falha sem chamar o datasource

  Cenário: consulta recebe ordenação determinística
    Dado um recurso com ordenação padrão por nome
    Quando o operador lista os registros sem escolher ordenação
    Então o datasource recebe a ordenação padrão seguida pelo identificador

  Cenário: identificador malicioso não chega ao datasource
    Dado uma requisição com resource, campo ou ordenação fora da allowlist
    Quando o operador tenta consultar o CRUD
    Então a requisição é recusada como inválida ou inexistente
    E o datasource não recebe a consulta

  Cenário: mutação e auditoria compartilham confirmação
    Dado uma criação autorizada em recurso mutável
    Quando a persistência e a auditoria concluem na mesma unidade de trabalho
    Então o registro e o evento de auditoria são confirmados
    E o hook posterior é executado somente após a confirmação

  Cenário: auditoria indisponível impede mutação
    Dado uma criação autorizada em recurso mutável
    Quando o audit sink falha na unidade de trabalho
    Então a operação retorna indisponibilidade temporária sem detalhe técnico
    E o hook posterior não é executado

  Cenário: adapter declarado conforme não atravessa o escopo
    Dado um adapter real exercitado pela suíte pública de conformidade
    Quando um registro criado no tenant A é buscado pelo tenant B
    Então o adapter retorna not_found sem revelar o registro

  Cenário: capability declarada tem comportamento verificável
    Dado um adapter que declara atomic_version e unit_of_work
    Quando duas alterações usam a mesma versão e uma transação falha
    Então exatamente uma alteração é confirmada
    E a mutação da transação falha é desfeita
