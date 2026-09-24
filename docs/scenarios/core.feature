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
