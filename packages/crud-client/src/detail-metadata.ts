import type { DetailDefinition, Field, FieldType, Value } from './types.js'

const fieldTypes = new Set<FieldType>(['string', 'text', 'integer', 'decimal', 'boolean', 'date', 'datetime', 'email', 'phone', 'enum', 'lookup'])

/** Resolves server-declared child metadata against the current parent values. */
export function resolveDetailFields(detail: DetailDefinition, parentFields: Record<string, Value>): Field[] {
  const fields = detail.Fields.map((field) => ({ ...field, Enum: field.Enum ? [...field.Enum] : field.Enum, Lookup: field.Lookup ? { ...field.Lookup, Dependencies: field.Lookup.Dependencies ? [...field.Lookup.Dependencies] : field.Lookup.Dependencies } : field.Lookup }))
  for (const source of detail.FieldMetadata ?? []) {
    const field = fields.find((candidate) => candidate.Key === source.Field)
    if (!field) continue
    const label = source.LabelField ? parentFields[source.LabelField] : undefined
    if (source.LabelField) {
      if (typeof label === 'string' && label.trim()) {
        field.Visible = true
        field.DisplayLabel = label.trim()
      } else {
        field.Visible = false
        field.DisplayLabel = undefined
      }
    }
    const type = source.TypeField ? parentFields[source.TypeField] : undefined
    if (typeof type === 'string' && fieldTypes.has(type.trim().toLowerCase() as FieldType)) {
      const nextType = type.trim().toLowerCase() as FieldType
      if (nextType !== field.Type) {
        field.Enum = undefined
        field.Lookup = undefined
        field.EnumControl = undefined
        field.Pattern = undefined
        field.PatternMessage = undefined
        field.Minimum = undefined
        field.Maximum = undefined
        field.MinLength = undefined
        field.MaxLength = undefined
      }
      field.Type = nextType
    }
    const required = source.RequiredField ? parentFields[source.RequiredField] : undefined
    if (typeof required === 'boolean') field.Required = required
    else if (typeof required === 'number' && (required === 0 || required === 1)) field.Required = required === 1
  }
  return fields
}
