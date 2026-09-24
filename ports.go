package crud

import (
	"context"
	"time"
)

// FilterOperator is an allowlisted server-side filter operation.
type FilterOperator string

const (
	FilterEqual          FilterOperator = "eq"
	FilterNotEqual       FilterOperator = "ne"
	FilterLessThan       FilterOperator = "lt"
	FilterLessOrEqual    FilterOperator = "lte"
	FilterGreaterThan    FilterOperator = "gt"
	FilterGreaterOrEqual FilterOperator = "gte"
	FilterContains       FilterOperator = "contains"
	FilterPrefix         FilterOperator = "prefix"
	FilterIsNull         FilterOperator = "is_null"
)

// Filter limits a query to an allowlisted field and operator.
type Filter struct {
	Field    FieldKey
	Operator FilterOperator
	Value    Value
}

// PageRequest contains exactly one paging strategy after validation.
type PageRequest struct {
	Mode   PageMode
	Number uint64
	Size   uint16
	Cursor string
}

// Query is the normalized server-side list request.
type Query struct {
	Search  string
	Filters []Filter
	Sort    []Sort
	Page    PageRequest
}

// Page is a paginated result returned by a DataSource.
type Page struct {
	Records    []Record
	NextCursor string
	Page       uint64
	Size       uint16
	Total      *uint64
}

// LookupQuery is the normalized request for a lookup field.
type LookupQuery struct {
	Search       string
	Cursor       string
	Size         uint16
	Dependencies Fields
}

// LookupOption is the only shape exposed by a lookup result.
type LookupOption struct {
	Value Value
	Label string
}

// LookupPage is a paginated set of lookup options.
type LookupPage struct {
	Options    []LookupOption
	NextCursor string
	Total      *uint64
}

// DataSource is the persistence port for a resource.
type DataSource interface {
	Capabilities(context.Context) Capabilities
	List(context.Context, Scope, Query) (Page, error)
	Get(context.Context, Scope, RecordID) (Record, error)
	Create(context.Context, Scope, Mutation) (Record, error)
	Update(context.Context, Scope, RecordID, Version, Mutation) (Record, error)
	Delete(context.Context, Scope, RecordID, Version, DeleteMode) error
	Lookup(context.Context, Scope, LookupQuery) (LookupPage, error)
}

// UnitOfWork creates the context used by persistence and mandatory audit in one
// atomic unit. A mutable v1 resource requires this guarantee.
type UnitOfWork interface {
	Within(context.Context, func(context.Context) error) error
}

// Principal identifies an authenticated actor without carrying authorization
// decisions or domain-specific profile data.
type Principal struct {
	ID   string
	Kind string
}

// PrincipalProvider resolves the trusted principal for an operation.
type PrincipalProvider interface {
	Principal(context.Context) (Principal, error)
}

// ScopeProvider resolves trusted scope for a registered resource.
type ScopeProvider interface {
	Scope(context.Context, ResourceKey) (Scope, error)
}

// Authorizer evaluates one resource action for a trusted principal.
// Current is nil for preliminary authorization and create operations.
type Authorizer interface {
	Authorize(context.Context, Principal, ResourceKey, Action, *Record) error
}

// AuditEvent records a mutation without exposing sensitive field values.
type AuditEvent struct {
	Resource      ResourceKey
	Action        Action
	RecordID      RecordID
	BeforeVersion Version
	AfterVersion  Version
	Principal     Principal
	Scope         Scope
	OccurredAt    time.Time
	CorrelationID string
}

// AuditSink persists a mutation audit event in the active UnitOfWork context.
type AuditSink interface {
	Append(context.Context, AuditEvent) error
}

// Translator resolves safe message codes for the active locale.
type Translator interface {
	Message(context.Context, MessageCode, map[string]any) string
}

// Clock makes time deterministic in the service and its tests.
type Clock interface {
	Now() time.Time
}
