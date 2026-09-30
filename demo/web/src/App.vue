<script setup lang="ts">
import { HttpCrudClient } from '@clear-platform-br/crud-client'
import { CrudScreen, ptBR } from '@clear-platform-br/crud-vue'
import DefinitionAccordion from './components/DefinitionAccordion.vue'
import { definitionSnippets } from './definitionSnippets'

const resource = 'clear_crud_minimal_demo_contacts'
const translatedResource = 'clear_crud_minimal_demo_contacts_ptbr'
const defaultsResource = 'clear_crud_minimal_demo_contacts_defaults'
const patternResource = 'clear_crud_minimal_demo_contacts_pattern'
const explicitOrderResource = 'clear_crud_minimal_demo_contacts_ordered'
const enumResource = 'clear_crud_disposable_junk_contacts_enum'
const enumControlsResource = 'clear_crud_enum_controls_demo'
const lookupResource = 'clear_crud_reference_lookup_demo'
const readModelResource = 'clear_crud_reference_read_model_demo'
const auxiliaryCatalogResource = 'clear_crud_auxiliary_catalogs'
const auxiliaryCatalogLookupResource = 'clear_crud_auxiliary_catalog_lookup_demo'
const auxiliaryCatalogChannelLookupResource = 'clear_crud_auxiliary_catalog_channel_lookup_demo'
const auxiliaryDetailLookupResource = 'clear_crud_auxiliary_catalog_detail_lookup_demo'
const client = new HttpCrudClient({ baseUrl: '/api/v1/crud' })

const labels: Record<string, string> = {
  [`crud.${resource}.title`]: 'Defaults automáticos — somente tabela',
  [`crud.${resource}.singular`]: 'Contato mínimo',
  [`crud.${translatedResource}.title`]: 'Situação traduzida — 10 por página',
  [`crud.${translatedResource}.singular`]: 'Contato mínimo',
  [`crud.${defaultsResource}.title`]: 'Defaults estáticos no formulário',
  [`crud.${defaultsResource}.singular`]: 'Contato com defaults',
  [`crud.${patternResource}.title`]: 'Validação declarativa de padrão',
  [`crud.${patternResource}.singular`]: 'Contato com nome validado',
  [`crud.${explicitOrderResource}.title`]: 'Ordem explícita — grid e formulário',
  [`crud.${explicitOrderResource}.singular`]: 'Contato mínimo ordenado',
  [`crud.${enumResource}.title`]: 'Contatos descartáveis — enum',
  [`crud.${enumResource}.singular`]: 'Contato descartável',
  [`crud.${enumControlsResource}.title`]: 'Controles de enum',
  [`crud.${enumControlsResource}.singular`]: 'Amostra de enum',
  [`crud.${lookupResource}.title`]: 'Lookup dependente — catálogo global compartilhado',
  [`crud.${lookupResource}.singular`]: 'Contato geográfico',
  [`crud.${readModelResource}.title`]: 'Modelo de leitura — join e campo calculado',
  [`crud.${readModelResource}.singular`]: 'Contato enriquecido',
  [`crud.${auxiliaryCatalogResource}.title`]: 'Tabela de Tabelas (TdT)',
  [`crud.${auxiliaryCatalogResource}.singular`]: 'Tabela da TdT',
  [`crud.${auxiliaryCatalogLookupResource}.title`]: 'Consumidor da TdT',
  [`crud.${auxiliaryCatalogLookupResource}.singular`]: 'Item da TdT',
  [`crud.${auxiliaryCatalogChannelLookupResource}.title`]: 'Lookup do catálogo Canais',
  [`crud.${auxiliaryCatalogChannelLookupResource}.singular`]: 'Uso do catálogo Canais',
  [`crud.${auxiliaryDetailLookupResource}.title`]: 'Lookup no detalhe da TdT',
  [`crud.${auxiliaryDetailLookupResource}.singular`]: 'Catálogo com lookup no detalhe',
  'crud.clear_crud_auxiliary_catalog_options.title': 'Itens da TdT',
  'crud.clear_crud_auxiliary_catalog_options.singular': 'Item da TdT',
  'crud.clear_crud_auxiliary_catalog_detail_lookup_options.title': 'Itens desta tabela',
  'crud.clear_crud_auxiliary_catalog_detail_lookup_options.singular': 'Item desta tabela',
  'crud.field.name': 'Nome',
  'crud.validation.name_uppercase': 'O nome deve começar com letra maiúscula.',
  'crud.field.notes': 'Observações',
  'crud.field.status': 'Situação',
  'crud.field.priority': 'Prioridade',
  'crud.field.channel': 'Canal',
  'crud.field.state': 'Estado',
	'crud.field.tone': 'Tom',
  'crud.status.new': 'Novo',
  'crud.status.review': 'Em análise',
  'crud.status.closed': 'Encerrado',
  'crud.priority.low': 'Baixa',
  'crud.priority.medium': 'Média',
  'crud.priority.high': 'Alta',
  'crud.channel.email': 'E-mail',
  'crud.channel.phone': 'Telefone',
  'crud.channel.whatsapp': 'WhatsApp',
  'crud.state.draft': 'Rascunho',
  'crud.state.active': 'Ativa',
  'crud.state.closed': 'Encerrada',
  'crud.tone.info': 'Informativo',
  'crud.tone.warning': 'Atenção',
  'crud.tone.urgent': 'Urgente',
  'crud.field.enabled': 'Ativo',
  'crud.field.zextra_01': 'Campo extra 01',
  'crud.field.zextra_02': 'Campo extra 02',
  'crud.field.zextra_03': 'Campo extra 03',
  'crud.field.zextra_04': 'Campo extra 04',
  'crud.field.zextra_05': 'Campo extra 05',
  'crud.field.zextra_06': 'Campo extra 06',
  'crud.field.zextra_07': 'Campo extra 07',
  'crud.field.zextra_08': 'Campo extra 08',
  'crud.field.zextra_09': 'Campo extra 09',
  'crud.field.zextra_10': 'Campo extra 10',
  'crud.field.zextra_11': 'Campo extra 11',
  'crud.field.zextra_12': 'Campo extra 12',
  'crud.field.zextra_13': 'Campo extra 13',
  'crud.field.city_id': 'Município',
  'crud.field.region_id': 'Região',
  'crud.field.state_id': 'Estado',
  'crud.field.state_name': 'Estado vindo do join',
  'crud.field.summary': 'Resumo calculado',
  'crud.field.calculated_number': 'Código numérico calculado',
  'crud.field.selected_option': 'Item da TdT',
  'crud.field.active': 'Ativo',
  'crud.field.code': 'Código do catálogo',
  'crud.field.management': 'Manutenção',
  'crud.field.max_options': 'Máximo de itens',
  'crud.field.title': 'Nome do catálogo',
  'crud.field.value_1_label': 'Campo 1 · rótulo',
  'crud.field.value_1_type': 'Campo 1 · tipo',
  'crud.field.value_1_required': 'Campo 1 · obrigatório',
  'crud.field.value_2_label': 'Campo 2 · rótulo',
  'crud.field.value_2_type': 'Campo 2 · tipo',
  'crud.field.value_2_required': 'Campo 2 · obrigatório',
  'crud.field.sequence': 'Sequência',
  'crud.field.sort_order': 'Ordem',
  'crud.field.value_1': 'Código / sigla',
  'crud.field.value_2': 'Nome da opção',
  'crud.catalog.management.fixed': 'Controlado pelo sistema',
  'crud.catalog.type.text': 'Texto',
  'crud.catalog.type.integer': 'Número inteiro',
  'crud.catalog.type.decimal': 'Número decimal',
  'crud.catalog.type.date': 'Data',
  'crud.catalog.type.boolean': 'Sim / não',
  'crud.catalog.type.lookup': 'Referência a outro cadastro',
  'crud.clear_crud_reference_values.title': 'Valores de referência',
  'crud.clear_crud_reference_values.singular': 'Valor de referência',
  'crud.clear_crud_reference_regions.title': 'Regiões de referência',
  'crud.clear_crud_reference_regions.singular': 'Região',
  'crud.clear_crud_reference_cities.title': 'Municípios de referência',
  'crud.clear_crud_reference_cities.singular': 'Município',
}

function translate(code: string): string | undefined {
  return labels[code]
}
</script>

<template>
  <main class="demo-page">
    <header class="demo-page-heading">
      <h1><span class="demo-smoke-label">Smoke 01 · definição mínima</span> Grid, formulário e defaults</h1>
      <p>Esta primeira grade foi registrada passando somente o nome da tabela; campos, tipos, busca, ordenação, paginação e formulário vêm dos defaults seguros.</p>
    </header>
    <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver a chamada que informa somente a tabela" :code="definitionSnippets.default" />
    <CrudScreen :resource="resource" :client="client" :messages="ptBR" :resolve-message="translate" />
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 02 · mesma tabela física</span> Exceções: 10 registros e situação traduzida</h2>
        <p>Esta definição reutiliza a tabela mínima, limita a paginação a 10 registros e apresenta os valores de <code>status</code> em português.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver paginação e enum traduzido" :code="definitionSnippets.translated" />
      <CrudScreen :resource="translatedResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke extra 01 · defaults estáticos</span> Valores iniciais declarados pelo consumidor</h2>
        <p>Esta definição reutiliza a mesma tabela e declara defaults estáticos para <code>status</code> e <code>enabled</code>. Eles aparecem no formulário de inclusão e também são aplicados pelo core quando a criação não envia esses campos.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver os defaults declarados" :code="definitionSnippets.defaults" />
      <CrudScreen :resource="defaultsResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke extra 02 · padrão declarativo</span> Validação de formato no contrato</h2>
        <p>Esta definição reutiliza a mesma tabela e declara uma expressão regular server-owned para o campo <code>name</code>. O core valida o valor e devolve a mensagem localizável quando o padrão não é atendido.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver o padrão e a mensagem declarados" :code="definitionSnippets.pattern" />
      <CrudScreen :resource="patternResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke extra 03 · ordem declarada</span> Projeção explícita no Grid e no Form</h2>
        <p>Esta definição reutiliza a mesma tabela física, mas declara outra ordem para as colunas. O catálogo canônico continua vindo do schema; a projeção explícita vence somente no Grid e no Form.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver Grid e Form na ordem declarada" :code="definitionSnippets.explicitOrder" />
      <CrudScreen :resource="explicitOrderResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 03 · tabela completa</span> Enum, archive e projeção declarada</h2>
        <p>Esta definição aponta para a mesma tabela, mas declara <code>status</code> como enum, mostra somente os campos selecionados e habilita o toggle compacto <code>Excluídos</code> para consultar registros arquivados como somente leitura.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver a exceção de enum e a projeção" :code="definitionSnippets.enum" />
      <CrudScreen :resource="enumResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 04 · três enums</span> Controles de enum escolhidos pelo programador</h2>
        <p>Esta tabela pequena tem quatro enums independentes: prioridade usa select, canal usa radio, estado usa segmented e tom usa buttons. Em viewport móvel, os quatro controles viram select nativo. A definição também usa <code>WithHardDelete()</code>, portanto a ação de apagar fisicamente aparece de propósito neste exemplo.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver os três controles declarados" :code="definitionSnippets.enumControls" />
      <CrudScreen :resource="enumControlsResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 05 · catálogo global</span> Lookups encadeados com cache server-side</h2>
        <p>Estado filtra Região e Região filtra Município. Os catálogos são globais e somente leitura; o cadastro do tenant apenas referencia seus IDs.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver recursos globais e dependências declaradas" :code="definitionSnippets.lookup" />
      <CrudScreen :resource="lookupResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 06 · filtro fixo server-owned</span> Estados em um catálogo compartilhado, particionado por tipo</h2>
        <p>O mesmo lookup lê a tabela global de valores de referência, mas a definição do consumidor fixa <code>Values: []crud.Value{&quot;state&quot;}</code>. Um valor aplica igualdade; vários aplicam pertencimento. Abra Estado e busque por <code>Real</code>: a moeda não aparece porque o navegador não controla esse filtro.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver FixedFilters declarado pelo consumidor" :code="definitionSnippets.fixedLookup" />
      <CrudScreen :resource="lookupResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 07 · Tabela de Tabelas (TdT) e mestre-detalhe</span> Tabelas da TdT e itens isolados por tabela</h2>
        <p>O contrato <code>clear.catalog.auxiliary.v1</code> descreve quatro posições de valor. Escolha uma tabela da TdT: somente os itens daquela tabela podem ser alterados.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver o vínculo mestre-detalhe genérico" :code="definitionSnippets.auxiliaryCatalog" />
      <CrudScreen :resource="auxiliaryCatalogResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 08 · consumidor da TdT</span> Outra tela usando itens de Estados e Territórios</h2>
        <p>Este CRUD não conhece a tabela de itens: recebe apenas o lookup registrado no servidor. A lista inclui itens ativos das tabelas 10 e 20, portanto os estados novos aparecem aqui sem alterar esta definição.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver o lookup fixo no catálogo auxiliar" :code="definitionSnippets.auxiliaryCatalogLookup" />
      <CrudScreen :resource="auxiliaryCatalogLookupResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 09 · lookup fixo da TdT</span> Canal vindo da Tabela de Tabelas</h2>
        <p>Esta definição reaproveita a tabela do quarto smoke, mas troca o enum de Canal pelo lookup da tabela Canais (30) na TdT. Ela aceita E-mail, Telefone e WhatsApp; não é um seletor de tabelas da TdT.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver o lookup fixo de Canais" :code="definitionSnippets.auxiliaryCatalogChannelLookup" />
      <CrudScreen :resource="auxiliaryCatalogChannelLookupResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 10 · modelo de leitura</span> Join e campo calculado na mesma grade</h2>
        <p>Esta definição é somente leitura: <code>state_name</code> vem do join com o catálogo global de estados, <code>summary</code> é calculado como texto e <code>calculated_number</code> é calculado numericamente pelo SQL server-owned do adapter. O tenant continua sendo aplicado pelo escopo confiável.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver o modelo de leitura registrado no adapter" :code="definitionSnippets.readModel" />
      <CrudScreen :resource="readModelResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
    <section class="demo-example">
      <header class="demo-page-heading">
        <h2><span class="demo-smoke-label">Smoke 11 · lookup no detalhe</span> Rótulo humano sem CRUD independente</h2>
        <p>O item filho é <code>detail-only</code>: não pode ser listado sozinho. Abra o catálogo <strong>Estados</strong> para ver o lookup resolver o código persistido para o nome da opção dentro do modal.</p>
      </header>
      <DefinitionAccordion title="Definição Go desta grade" subtitle="Abra para ver o lookup declarado no filho detail-only" :code="definitionSnippets.auxiliaryDetailLookup" />
      <CrudScreen :resource="auxiliaryDetailLookupResource" :client="client" :messages="ptBR" :resolve-message="translate" />
    </section>
  </main>
</template>

<style scoped>
.demo-page { min-height: 100vh; padding: 1.5rem; background: #f7f6f2; color: #1f2933; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", ui-sans-serif, sans-serif; font-size: 1rem; }
.demo-page-heading { max-width: 72rem; margin: 0 auto 1rem; }
.demo-page-heading h1,
.demo-page-heading h2 { display: flex; flex-wrap: wrap; align-items: baseline; gap: 0.35rem 0.7rem; margin: 0.2rem 0 0.65rem; color: #18312d; font-weight: 750; line-height: 1.18; }
.demo-page-heading h1 { font-size: clamp(1.65rem, 2.5vw, 2.35rem); }
.demo-page-heading h2 { font-size: clamp(1.45rem, 2.15vw, 1.95rem); }
.demo-page-heading p { max-width: 70rem; margin: 0; color: #52615e; font-size: 1.04rem; line-height: 1.55; }
.demo-example { max-width: 72rem; margin: 3rem auto 0; }
.demo-example code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.92em; }
.demo-smoke-label { color: #1c776b; font-size: clamp(0.95rem, 1.25vw, 1.08rem); font-weight: 800; letter-spacing: 0.025em; white-space: nowrap; }
:global(body:has([data-clear-crud-theme="dark"])) .demo-page { background: #15191e; color: #edf2f7; }
:global(body:has([data-clear-crud-theme="dark"])) .demo-page-heading h1 { color: #edf2f7; }
:global(body:has([data-clear-crud-theme="dark"])) .demo-page-heading h2 { color: #edf2f7; }
:global(body:has([data-clear-crud-theme="dark"])) .demo-page-heading p { color: #b7c2cd; }
:global(body:has([data-clear-crud-theme="dark"])) .demo-smoke-label { color: #6ed7c4; }
:global(body:has([data-clear-crud-theme="dark"])) .demo-example code { color: #c9f5e7; }
</style>
