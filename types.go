// Package crud provides the public, domain-neutral types for clear.crud.
package crud

// ResourceKey identifies a resource registered by the server.
// It is never a database table, collection, or client-provided SQL identifier.
type ResourceKey string

// FieldKey identifies a field declared by a resource definition.
type FieldKey string

// RecordID is the opaque public identity of a record.
// Adapters resolve it to their physical identity format.
type RecordID string

// Version is the optimistic concurrency version of a mutable record.
type Version uint64

// MessageCode identifies a localizable, safe message.
type MessageCode string

// Action identifies an operation that can be authorized independently.
type Action string

const (
	// ActionCreate authorizes record creation.
	ActionCreate Action = "create"
	// ActionRead authorizes definition, list, get, and lookup reads.
	ActionRead Action = "read"
	// ActionUpdate authorizes record updates.
	ActionUpdate Action = "update"
	// ActionDelete authorizes archive or physical deletion according to policy.
	ActionDelete Action = "delete"
	// ActionHelp authorizes contextual help when a host protects it.
	ActionHelp Action = "help"
)

// FieldType is the normalized public type of a field.
type FieldType string

const (
	FieldString   FieldType = "string"
	FieldText     FieldType = "text"
	FieldInteger  FieldType = "integer"
	FieldDecimal  FieldType = "decimal"
	FieldBoolean  FieldType = "boolean"
	FieldDate     FieldType = "date"
	FieldDateTime FieldType = "datetime"
	FieldEmail    FieldType = "email"
	FieldPhone    FieldType = "phone"
	FieldEnum     FieldType = "enum"
	FieldLookup   FieldType = "lookup"
)

// Value is normalized by the core before it reaches a DataSource.
// Version 1 accepts nil, string, bool, and int64. Decimal, date, and datetime
// values use canonical strings. Objects, slices, and floating-point values are
// rejected by validation added in a later PR.
type Value = any

// Fields contains public field values keyed by a registered FieldKey.
type Fields map[FieldKey]Value

// Scope contains trusted, host-provided boundaries such as tenant_id.
// Clients never choose these values.
type Scope map[string]string

// Record is the public representation returned by a DataSource.
type Record struct {
	ID      RecordID
	Version Version
	Fields  Fields
}

// Mutation contains only writable field values after core validation.
type Mutation struct {
	Fields Fields
}

// DeleteMode describes the physical behavior behind the public delete action.
type DeleteMode string

const (
	DeleteModeNone DeleteMode = "none"
	// DeleteModeArchive preserves the record as archived. It is the v1 soft-delete policy.
	DeleteModeArchive DeleteMode = "archive"
	// DeleteModeSoftDelete is the explicit name for the archive policy.
	// It has the same wire value to keep v1 definitions compatible.
	DeleteModeSoftDelete DeleteMode = DeleteModeArchive
	DeleteModeHardDelete DeleteMode = "hard_delete"
)

// Capability describes a verifiable guarantee offered by a DataSource.
type Capability string

const (
	CapabilityTotalCount     Capability = "total_count"
	CapabilityOffsetPage     Capability = "offset_page"
	CapabilityCursorPage     Capability = "cursor_page"
	CapabilityAtomicVersion  Capability = "atomic_version"
	CapabilityUnitOfWork     Capability = "unit_of_work"
	CapabilityArchive        Capability = "archive"
	CapabilityHardDelete     Capability = "hard_delete"
	CapabilityContainsSearch Capability = "contains_search"
	CapabilityPrefixSearch   Capability = "prefix_search"
	CapabilityLookup         Capability = "lookup"
)

// Capabilities is a set of guarantees declared by a DataSource.
type Capabilities map[Capability]struct{}

// Has reports whether the capability is declared.
func (capabilities Capabilities) Has(capability Capability) bool {
	_, ok := capabilities[capability]
	return ok
}
