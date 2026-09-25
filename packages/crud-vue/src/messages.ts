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
  cardinality: string
  invalid: string
  length: string
  range: string
}

export const ptBR: CrudMessages = {
  create: 'Incluir', edit: 'Editar', remove: 'Apagar', help: 'Ajuda', search: 'Buscar', searchPlaceholder: 'Buscar registros',
  cancel: 'Cancelar', save: 'Salvar', confirmRemove: 'Apagar registro definitivamente?', confirmRemoveBody: 'Esta ação não pode ser desfeita.', archive: 'Arquivar', confirmArchive: 'Arquivar registro?', confirmArchiveBody: 'O registro deixará de aparecer nas listas usuais e poderá ser recuperado conforme a política do sistema.',
  noRecords: 'Nenhum registro encontrado.', previousPage: 'Página anterior', nextPage: 'Próxima página', loading: 'Carregando…', close: 'Fechar',
  addItem: 'Adicionar item', removeItem: 'Remover item', required: 'Obrigatório.', validation: 'Revise os campos destacados.',
  saved: 'Alterações salvas.', removed: 'Registro removido.', unavailable: 'Não foi possível concluir a ação agora.', cardinality: 'A quantidade de itens não é válida.',
  invalid: 'Valor inválido.', length: 'O tamanho informado não é permitido.', range: 'O valor está fora do limite permitido.',
}

export const enUS: CrudMessages = {
  create: 'Add', edit: 'Edit', remove: 'Delete', help: 'Help', search: 'Search', searchPlaceholder: 'Search records',
  cancel: 'Cancel', save: 'Save', confirmRemove: 'Delete record permanently?', confirmRemoveBody: 'This action cannot be undone.', archive: 'Archive', confirmArchive: 'Archive record?', confirmArchiveBody: 'The record will leave the usual lists and may be recovered according to system policy.',
  noRecords: 'No records found.', previousPage: 'Previous page', nextPage: 'Next page', loading: 'Loading…', close: 'Close',
  addItem: 'Add item', removeItem: 'Remove item', required: 'Required.', validation: 'Review the highlighted fields.',
  saved: 'Changes saved.', removed: 'Record removed.', unavailable: 'We could not complete this action now.', cardinality: 'The number of items is invalid.',
  invalid: 'Invalid value.', length: 'The entered length is not allowed.', range: 'The value is outside the allowed limit.',
}

export type Translate = (message: string) => string

export function createTranslator(messages: CrudMessages = ptBR, resolve?: (code: string) => string | undefined): Translate {
  const catalog: Record<string, string> = {
    'crud.ui.validation': messages.validation,
    'crud.ui.saved': messages.saved,
    'crud.ui.removed': messages.removed,
    'crud.ui.unavailable': messages.unavailable,
    'crud.field.required': messages.required,
    'crud.field.invalid': messages.invalid,
    'crud.field.length': messages.length,
    'crud.field.range': messages.range,
    'crud.detail.cardinality': messages.cardinality,
  }
  return (message) => resolve?.(message) ?? catalog[message] ?? message
}
