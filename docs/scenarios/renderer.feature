Funcionalidade: Renderer padrão de CRUD

  Cenário: Operador inclui um registro com destinos filhos declarados
    Dado que o operador pode criar o recurso pai e seus destinos filhos
    E que o formulário declara a coleção filha com mínimo e máximo permitidos
    Quando o operador informa os campos do pai e os destinos
    Então o renderer envia uma única mutation explícita
    E o campo interno de vínculo com o pai não aparece nem é enviado
    E o resultado confirmado é apresentado sem mensagem técnica

  Cenário: Falha de validação preserva o rascunho
    Dado que o operador está editando um registro
    Quando o servidor recusa um campo por validação
    Então o renderer mantém os valores informados
    E apresenta o erro associado ao campo
    E não expõe detalhes técnicos do servidor

  Cenário: Editor identifica o registro pai selecionado
    Dado que a definição declara `Presentation.TitleField` para um campo visível
    Quando o operador abre um registro pai existente para editar seus itens
    Então o cabeçalho do modal mostra o valor desse campo
    E a inclusão de um novo pai continua usando o título fixo da definição
    E a opção não altera escopo, autorização, mutation ou persistência

  Cenário: Campo e subtotal formatam número sem converter o dado em texto
    Dado que o consumidor declara locale, moeda e escala para um campo e sua medida
    Quando a grade mostra o campo e o subtotal
    Então ambos são localizados somente na apresentação
    E os valores numéricos originais permanecem inalterados para contas e transporte

  Cenário: Host abre detalhes sem conceder edição pelo renderer
    Dado que o host declara um rótulo para a ação de detalhes
    Quando o operador aciona detalhes em uma linha
    Então o renderer emite somente o ID do registro ao host
    E não emite uma mutation de inclusão, edição ou remoção

  Cenário: Ações simbólicas continuam acessíveis
    Dado que a lista apresenta registros editáveis e apagáveis
    Quando o operador usa teclado ou leitor de tela
    Então incluir é identificado por "+"
    E editar é identificado por "lápis"
    E apagar é identificado por "X" e pede confirmação
