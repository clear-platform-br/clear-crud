package crud

import "context"

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
	Keys []string
}

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
	PageSize     uint16
}

// Field describes a resource field without exposing physical storage details.
type Field struct {
	Key       FieldKey
	Label     MessageCode
	Help      MessageCode
	Type      FieldType
	Required  bool
	ReadOnly  bool
	Visible   bool
	Sensitive bool
	MinLength int
	MaxLength int
	Minimum   string
	Maximum   string
	Enum      []Option
	Lookup    *LookupDefinition
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

// ListDefinition describes the server-side collection view.
type ListDefinition struct {
	Columns     []FieldKey
	Searchable  []FieldKey
	Sortable    []FieldKey
	DefaultSort []Sort
	Pagination  PaginationDefinition
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
	Contract     ContractVersion
	Key          ResourceKey
	Labels       Labels
	Scope        ScopeRequirements
	Permissions  Permissions
	Fields       []Field
	List         ListDefinition
	Presentation Presentation
	Source       DataSource
	UOW          UnitOfWork
	Delete       DeletePolicy
	Concurrency  ConcurrencyPolicy
	Validator    Validator
	Hooks        Hooks
}
