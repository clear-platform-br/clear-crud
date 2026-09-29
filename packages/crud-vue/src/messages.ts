export interface CrudMessages {
  create: string
  edit: string
  remove: string
  help: string
  search: string
  searchPlaceholder: string
  cancel: string
  save: string
  confirmRemove: string
  confirmRemoveBody: string
  archive: string
  confirmArchive: string
  confirmArchiveBody: string
  noRecords: string
  previousPage: string
  nextPage: string
  loading: string
  close: string
  addItem: string
  removeItem: string
  required: string
  validation: string
  saved: string
  removed: string
  unavailable: string
  deleteRestricted?: string
  cardinality: string
  invalid: string
  pattern?: string
  length: string
  range: string
  lightMode?: string
  darkMode?: string
  compactDensity?: string
  comfortableDensity?: string
  includeArchived?: string
  includeArchivedHelp?: string
  archived?: string
  filters?: string
  filterHint?: string
  filterAll?: string
  applyFilters?: string
  clearFilters?: string
  booleanTrue?: string
  booleanFalse?: string
  selectOption?: string
  lookupOptions?: string
  lookupDependencyRequired?: string
  lookupNoOptions?: string
  lookupSearchHint?: string
  lookupSearchMinimum?: string
}

export const ptBR: CrudMessages = {
  create: 'Incluir', edit: 'Editar', remove: 'Apagar', help: 'Ajuda', search: 'Buscar', searchPlaceholder: 'Buscar registros',
  cancel: 'Cancelar', save: 'Salvar', confirmRemove: 'Apagar registro definitivamente?', confirmRemoveBody: 'Esta ação não pode ser desfeita.', archive: 'Arquivar', confirmArchive: 'Arquivar registro?', confirmArchiveBody: 'O registro deixará de aparecer nas listas usuais.',
  noRecords: 'Nenhum registro encontrado.', previousPage: 'Página anterior', nextPage: 'Próxima página', loading: 'Carregando…', close: 'Fechar',
  addItem: 'Adicionar item', removeItem: 'Remover item', required: 'Obrigatório.', validation: 'Revise os campos destacados.',
  saved: 'Alterações salvas.', removed: 'Registro removido.', unavailable: 'Não foi possível concluir a ação agora.', deleteRestricted: 'Não é possível apagar: existem registros dependentes ou a política não permite.', cardinality: 'A quantidade de itens não é válida.',
  invalid: 'Valor inválido.', pattern: 'O formato informado não é permitido.', length: 'O tamanho informado não é permitido.', range: 'O valor está fora do limite permitido.',
  lightMode: 'Usar modo claro', darkMode: 'Usar modo escuro', compactDensity: 'Usar linhas densas', comfortableDensity: 'Usar linhas confortáveis', includeArchived: 'Excluídos', includeArchivedHelp: 'Incluir registros excluídos', archived: 'Excluído', filters: 'Filtros', filterHint: 'Filtre cada coluna sem alterar a definição.', filterAll: 'Todos', applyFilters: 'Aplicar filtros', clearFilters: 'Limpar filtros', booleanTrue: 'Sim', booleanFalse: 'Não', selectOption: 'Selecione…', lookupOptions: 'Abrir opções', lookupDependencyRequired: 'Escolha uma opção primeiro.', lookupNoOptions: 'Nenhuma opção disponível.', lookupSearchHint: 'A lista inicial mostra até {size} opções. Digite para buscar outras.', lookupSearchMinimum: 'Digite pelo menos {size} caracteres para pesquisar.',
}

export const enUS: CrudMessages = {
  create: 'Add', edit: 'Edit', remove: 'Delete', help: 'Help', search: 'Search', searchPlaceholder: 'Search records',
  cancel: 'Cancel', save: 'Save', confirmRemove: 'Delete record permanently?', confirmRemoveBody: 'This action cannot be undone.', archive: 'Archive', confirmArchive: 'Archive record?', confirmArchiveBody: 'The record will no longer appear in normal lists.',
  noRecords: 'No records found.', previousPage: 'Previous page', nextPage: 'Next page', loading: 'Loading…', close: 'Close',
  addItem: 'Add item', removeItem: 'Remove item', required: 'Required.', validation: 'Review the highlighted fields.',
  saved: 'Changes saved.', removed: 'Record removed.', unavailable: 'We could not complete this action now.', deleteRestricted: 'This record cannot be deleted because dependent records exist or policy disallows it.', cardinality: 'The number of items is invalid.',
  invalid: 'Invalid value.', pattern: 'The entered format is not allowed.', length: 'The entered length is not allowed.', range: 'The value is outside the allowed limit.',
  lightMode: 'Use light mode', darkMode: 'Use dark mode', compactDensity: 'Use compact rows', comfortableDensity: 'Use comfortable rows', includeArchived: 'Deleted', includeArchivedHelp: 'Include deleted records', archived: 'Deleted', filters: 'Filters', filterHint: 'Filter each column without changing the definition.', filterAll: 'All', applyFilters: 'Apply filters', clearFilters: 'Clear filters', booleanTrue: 'Yes', booleanFalse: 'No', selectOption: 'Select…', lookupOptions: 'Open options', lookupDependencyRequired: 'Choose an option first.', lookupNoOptions: 'No options available.', lookupSearchHint: 'The initial list shows up to {size} options. Type to search for more.', lookupSearchMinimum: 'Type at least {size} characters to search.',
}

export type Translate = (message: string) => string

export function createTranslator(messages: CrudMessages = ptBR, resolve?: (code: string) => string | undefined): Translate {
  const catalog: Record<string, string> = {
    'crud.ui.validation': messages.validation,
    'crud.ui.saved': messages.saved,
    'crud.ui.removed': messages.removed,
    'crud.ui.unavailable': messages.unavailable,
    'crud.error.delete_restricted': messages.deleteRestricted ?? messages.unavailable,
    'crud.error.invalid_request': messages.invalid,
    'crud.error.validation_failed': messages.validation,
    'crud.field.required': messages.required,
    'crud.field.invalid': messages.invalid,
    'crud.field.pattern': messages.pattern ?? messages.invalid,
    'crud.field.length': messages.length,
    'crud.field.range': messages.range,
    'crud.detail.cardinality': messages.cardinality,
  }
  return (message) => resolve?.(message) ?? catalog[message] ?? message
}
