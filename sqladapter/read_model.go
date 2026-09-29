package sqladapter

import (
	"context"
	"database/sql"
	"fmt"

	crud "github.com/clear-platform-br/clear-crud"
)

const readModelAlias = "clear_crud_read_model"

// ReadModelConfig describes a server-owned SQL read model. Query is supplied
// only during bootstrap by the adapter consumer; it is never accepted from
// HTTP. The query must expose IDColumn, every ScopeColumns alias and every
// Fields alias as result columns.
type ReadModelConfig struct {
	Query        string
	IDColumn     Identifier
	ScopeColumns map[string]Identifier
	Fields       map[crud.FieldKey]Identifier
	Searchable   []crud.FieldKey
}

type readModelField struct {
	key    crud.FieldKey
	column Identifier
}

type readModelScope struct {
	key    string
	column Identifier
}

// ReadModelSource executes a bounded, read-only SQL read model. The core
// still owns field, filter, sort and scope validation; this adapter owns the
// database-specific query that produces the model.
type ReadModelSource struct {
	db         *sql.DB
	query      string
	idColumn   Identifier
	scopes     []readModelScope
	fields     []readModelField
	fieldByKey map[crud.FieldKey]Identifier
	searchable []crud.FieldKey
	selectList string
}

// NewReadModel validates a server-owned query and verifies its result aliases
// during bootstrap. It does not create or alter database objects.
func NewReadModel(ctx context.Context, db *sql.DB, config ReadModelConfig) (*ReadModelSource, error) {
	if db == nil {
		return nil, fmt.Errorf("sqladapter: nil database")
	}
	prepared, err := prepareReadModel(ctx, db, config)
	if err != nil {
		return nil, err
	}
	return &ReadModelSource{
		db:         db,
		query:      prepared.query,
		idColumn:   config.IDColumn,
		scopes:     prepared.scopes,
		fields:     prepared.fields,
		fieldByKey: prepared.fieldByKey,
		searchable: append([]crud.FieldKey(nil), config.Searchable...),
		selectList: prepared.selectList,
	}, nil
}

// RegisterReadModel registers a read-only definition backed by a server-owned
// adapter query. The definition supplies public labels, fields and grid
// allowlists; the source supplies the joined or calculated values.
func RegisterReadModel(ctx context.Context, db *sql.DB, registry *crud.Registry, definition crud.Definition, config ReadModelConfig) error {
	if registry == nil {
		return fmt.Errorf("sqladapter: nil registry")
	}
	if definition.Source != nil {
		return fmt.Errorf("sqladapter: read model definition must not provide a source")
	}
	if definition.Permissions.Create != "" || definition.Permissions.Update != "" || definition.Permissions.Delete != "" || definition.Delete.Mode != crud.DeleteModeNone {
		return fmt.Errorf("sqladapter: read models are read-only")
	}
	definition = completeReadModelDefinition(definition)
	completed, err := completeReadModelConfig(definition, config)
	if err != nil {
		return err
	}
	source, err := NewReadModel(ctx, db, completed)
	if err != nil {
		return err
	}
	definition.Source = source
	definition.UOW = nil
	definition.Concurrency.Mode = crud.ConcurrencyNone
	return registry.Register(ctx, definition)
}

// Capabilities reports the guarantees provided by the read model.
func (source *ReadModelSource) Capabilities(context.Context) crud.Capabilities {
	return crud.Capabilities{
		crud.CapabilityTotalCount:     {},
		crud.CapabilityOffsetPage:     {},
		crud.CapabilityContainsSearch: {},
	}
}

// Metadata marks every field returned by a read model as read-only. The
// definition remains responsible for labels, types and the grid allowlist.
func (source *ReadModelSource) Metadata(context.Context) (crud.SourceMetadata, error) {
	if source == nil {
		return crud.SourceMetadata{}, fmt.Errorf("sqladapter: read model source is not initialized")
	}
	fields := make(map[crud.FieldKey]crud.FieldMetadata, len(source.fields))
	for _, field := range source.fields {
		fields[field.key] = crud.FieldMetadata{ReadOnly: true}
	}
	return crud.SourceMetadata{Fields: fields}, nil
}

// Read models never accept mutations or dynamic lookup definitions.
func (*ReadModelSource) Create(context.Context, crud.Scope, crud.Mutation) (crud.Record, error) {
	return crud.Record{}, public(crud.ErrorInvalidRequest)
}

func (*ReadModelSource) Update(context.Context, crud.Scope, crud.RecordID, crud.Version, crud.Mutation) (crud.Record, error) {
	return crud.Record{}, public(crud.ErrorInvalidRequest)
}

func (*ReadModelSource) Delete(context.Context, crud.Scope, crud.RecordID, crud.Version, crud.DeleteMode) error {
	return public(crud.ErrorDeleteRestricted)
}

func (*ReadModelSource) Lookup(context.Context, crud.Scope, crud.LookupQuery) (crud.LookupPage, error) {
	return crud.LookupPage{}, public(crud.ErrorInvalidRequest)
}
