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
