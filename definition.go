package crud

import (
	"context"
	"regexp"
)

// ContractVersion identifies a public clear.crud contract.
type ContractVersion string

const (
	// ContractDefinitionV1 is the public resource-definition contract.
	ContractDefinitionV1 ContractVersion = "clear.crud.definition.v1"
)

// Labels contains safe, localizable labels for a resource.
type Labels struct {
	Title    MessageCode
	Singular MessageCode
	Help     MessageCode
}

// Permissions maps each resource action to a host-defined permission.
// An empty permission makes its action unavailable.
type Permissions struct {
	Create string
	Read   string
	Update string
	Delete string
	Help   string
}

// ScopeRequirements declares trusted scope keys required by a resource.
type ScopeRequirements struct {
	Mode ScopeMode
	Keys []string
}

// ScopeMode identifies whether a resource is tenant-scoped or an explicitly
// global reference resource. Global resources are read-only and never inherit
// a tenant boundary accidentally.
type ScopeMode string

const (
	ScopeModeTenant ScopeMode = "tenant"
	ScopeModeGlobal ScopeMode = "global"
)

// ResourceAccessMode controls whether a registered resource has independent
// CRUD routes or can only be reached through a declared master-detail parent.
// The empty value preserves the standalone default for existing definitions.
type ResourceAccessMode string

const (
	ResourceAccessStandalone ResourceAccessMode = "standalone"
	ResourceAccessDetailOnly ResourceAccessMode = "detail_only"
)

// Option is a fixed enum option.
type Option struct {
	Value Value
	Label MessageCode
}

// LookupDefinition describes a registered, server-side lookup.
type LookupDefinition struct {
	Resource     ResourceKey
	ValueField   FieldKey
	LabelField   FieldKey
	Dependencies []FieldKey
	// FixedFilters limits the target lookup to registered equality or membership
	// predicates. It is never populated from a transport request and does not
	// replace trusted scope, authorization, or archive behavior.
	FixedFilters []FixedLookupFilter
	PageSize     uint16
	// MinSearchLength is the minimum number of characters a renderer should
	// require before issuing a filtered lookup request. Zero uses the safe
	// default of three characters; the explicit setting is an exception for
	// catalogs whose identifiers are intentionally shorter.
	MinSearchLength uint16
}

// FixedLookupFilter is one server-owned equality or membership predicate for a
// lookup target. One value means equality; multiple values mean membership.
// It is intentionally narrower than Filter: lookup declarations do not carry
// operators or SQL fragments.
type FixedLookupFilter struct {
	Field  FieldKey
	Values []Value
}

func cloneFixedLookupFilters(filters []FixedLookupFilter) []FixedLookupFilter {
	clone := make([]FixedLookupFilter, len(filters))
	for index, filter := range filters {
		clone[index] = filter
		clone[index].Values = append([]Value(nil), filter.Values...)
	}
	return clone
}

// BooleanDisplay defines the two symbols used by a renderer for a boolean
// field. Symbols are presentation metadata, not stored values or HTML.
type BooleanDisplay struct {
	True  string
	False string
}

// EnumControl selects the semantic input used to choose one enum value.
// Buttons are separate actions; segmented keeps adjacent actions visually joined.
// Auto lets the renderer adapt to option count and viewport size.
type EnumControl string

const (
	EnumControlAuto      EnumControl = "auto"
	EnumControlSelect    EnumControl = "select"
	EnumControlRadio     EnumControl = "radio"
	EnumControlSegmented EnumControl = "segmented"
	EnumControlButtons   EnumControl = "buttons"
)

// Field describes a resource field without exposing physical storage details.
type Field struct {
	Key   FieldKey
	Label MessageCode
	// DisplayLabel is an optional server-resolved literal label. It is used
	// when a registered parent record supplies presentation metadata for an
	// existing child field; static definitions continue to use Label.
	DisplayLabel string `json:",omitempty"`
	Help         MessageCode
	Type         FieldType
	// Required is preserved for compatibility with definition.v1 consumers.
	// New definitions are required by default; use Optional for the exception.
	Required bool
	// Optional allows an empty value. Fields are required unless Optional is true.
	Optional bool `json:"-"`
	// ReadOnly fields may be populated by a server-owned read projection,
	// including a joined or calculated display value. They are exposed to the
	// renderer but are never accepted in mutations.
	ReadOnly bool
	// CreateOnly accepts a value on create and preserves it on update. It is
	// useful for immutable business keys such as an apartment number.
	CreateOnly bool
	Visible    bool
	Sensitive  bool
	// Default is a static value used for a new record. It is exposed to the
	// renderer for a visible create form and applied by the core even when the
	// field is hidden. Hidden defaults are useful for server-owned technical
	// fields; dynamic defaults remain adapter-owned.
	Default        Value           `json:",omitempty"`
	BooleanDisplay *BooleanDisplay `json:",omitempty"`
	EnumControl    EnumControl     `json:",omitempty"`
	// Pattern is a server-owned RE2 expression applied to string-backed values.
	// The expression is declarative metadata; it is never accepted from HTTP.
	Pattern string `json:",omitempty"`
	// PatternMessage is returned when Pattern rejects a non-empty value. When it
	// is empty, the renderer uses the generic crud.field.pattern message code.
	PatternMessage MessageCode `json:",omitempty"`
	MinLength      int
	MaxLength      int
	Minimum        string
	Maximum        string
	Enum           []Option
	Lookup         *LookupDefinition
}

// IsRequired reports the canonical requiredness of a field.
func (field Field) IsRequired() bool {
	return !field.Optional
}

// PageMode determines how a resource pages its result set.
type PageMode string

const (
	PageModeOffset PageMode = "offset"
	PageModeCursor PageMode = "cursor"
)

// PaginationDefinition constrains page behavior for a resource.
type PaginationDefinition struct {
	Mode         PageMode
	DefaultSize  uint16
	AllowedSizes []uint16
	Total        bool
}

const (
	maxPageSize         uint16 = 100
	maxAllowedPageSizes        = 8
)

// SortDirection determines the order of a sortable field.
type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

// Sort describes one public ordering term.
type Sort struct {
	Field     FieldKey
	Direction SortDirection
}

// ArchiveVisibility controls whether a Grid may request archived records in
// addition to active records. The default is active-only.
type ArchiveVisibility string

const (
	ArchiveVisibilityActiveOnly        ArchiveVisibility = "active_only"
	ArchiveVisibilityActiveAndArchived ArchiveVisibility = "active_and_archived"
)

// GridDefinition describes the server-side grid view. Columns may include
// visible read-only projection fields returned by a registered DataSource.
type GridDefinition struct {
	Columns           []FieldKey
	Searchable        []FieldKey
	Sortable          []FieldKey
	DefaultSort       []Sort
	Pagination        PaginationDefinition
	ArchiveVisibility ArchiveVisibility `json:",omitempty"`
}

// FormDefinition describes the server-owned field projection and order of a
// CRUD form. An empty Fields slice means all visible fields in Definition.Fields
// order; field metadata remains canonical in Definition.Fields.
type FormDefinition struct {
	Fields []FieldKey
}

// CollectionMode is the preferred collection presentation for a renderer.
type CollectionMode string

const (
	CollectionAuto  CollectionMode = "auto"
	CollectionTable CollectionMode = "table"
	CollectionCards CollectionMode = "cards"
	CollectionList  CollectionMode = "list"
)

// Density is the preferred visual density for a renderer.
type Density string

const (
	DensityCompact     Density = "compact"
	DensityComfortable Density = "comfortable"
)

// Presentation contains semantic renderer hints, never CSS or component names.
type Presentation struct {
	Collection CollectionMode
	Density    Density
	// TitleField identifies the non-sensitive field used to contextualize an
	// existing record editor. Empty keeps the definition label as the title.
	TitleField FieldKey `json:",omitempty"`
}

// ConcurrencyMode controls write-conflict behavior.
type ConcurrencyMode string

const (
	ConcurrencyNone    ConcurrencyMode = "none"
	ConcurrencyVersion ConcurrencyMode = "version"
)

// ConcurrencyPolicy configures a resource's optimistic concurrency behavior.
type ConcurrencyPolicy struct {
	Mode ConcurrencyMode
}

// DeletePolicy configures the physical effect behind the public delete action.
type DeletePolicy struct {
	Mode DeleteMode
}

// DetailFieldMetadataSource maps optional parent fields to metadata overrides
// for one already-declared child field. It never creates fields or changes the
// child storage contract. The mapping is server-owned and the browser cannot
// provide it.
type DetailFieldMetadataSource struct {
	Field         FieldKey
	LabelField    FieldKey `json:",omitempty"`
	TypeField     FieldKey `json:",omitempty"`
	RequiredField FieldKey `json:",omitempty"`
}

// DetailParentAccess limits a child collection to parent records whose field
// contains one of the declared server-owned values. It is evaluated by the
// core for detail reads and mutations; clients never provide the condition.
type DetailParentAccess struct {
	Field  FieldKey
	Values []Value
}

// DetailDefinition declares one server-registered child collection. ParentField
// is internal: the service supplies the parent record ID and rejects client input.
type DetailDefinition struct {
	Key           DetailKey
	Resource      ResourceKey
	ParentField   FieldKey
	Minimum       uint16
	Maximum       uint16
	AllowCreate   bool
	AllowUpdate   bool
	AllowDelete   bool
	FieldMetadata []DetailFieldMetadataSource
	ParentAccess  *DetailParentAccess
}

// FieldErrors maps a field to a safe, localizable validation message.
type FieldErrors map[FieldKey]MessageCode

// Validator evaluates resource-specific invariants after structural validation.
type Validator func(context.Context, Scope, Mutation) FieldErrors

// MutationEvent is passed to a hook after a successful commit.
type MutationEvent struct {
	Action Action
	Record Record
}

// Hooks provides typed lifecycle extension points. Hooks never choose scope,
// generate SQL, or return technical messages to operators.
type Hooks struct {
	BeforeValidate func(context.Context, Mutation) (Mutation, error)
	BeforeCreate   func(context.Context, Scope, Mutation) error
	BeforeUpdate   func(context.Context, Scope, Record, Mutation) error
	BeforeDelete   func(context.Context, Scope, Record) error
	AfterCommit    func(context.Context, MutationEvent) error
}

// Definition is an immutable server-registered CRUD resource.
type Definition struct {
	Contract      ContractVersion
	Key           ResourceKey
	Labels        Labels
	Scope         ScopeRequirements
	Permissions   Permissions
	Access        ResourceAccessMode
	Fields        []Field
	Details       []DetailDefinition
	Grid          GridDefinition
	Form          FormDefinition
	Presentation  Presentation
	Source        DataSource
	UOW           UnitOfWork
	Delete        DeletePolicy
	Concurrency   ConcurrencyPolicy
	Validator     Validator
	Hooks         Hooks
	fieldPatterns map[FieldKey]*regexp.Regexp
}
