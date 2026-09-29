package sqladapter

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	crud "github.com/clear-platform-br/clear-crud"
)

const (
	defaultAutoTextMaxLength = 255
	maximumAutoTextMaxLength = 65535
	maximumAutoPatternLength = 512
	defaultAutoGridColumns   = 8
	defaultAutoGridPageSize  = 25
)

// FieldOverride is an explicit, server-owned exception to an automatic field
// default. It belongs in the consumer's registration call, never in schema
// name heuristics or HTTP input.
type FieldOverride struct {
	MaxLength      int
	Optional       bool
	Default        crud.Value
	BooleanDisplay *crud.BooleanDisplay
	Enum           []crud.Option
	EnumControl    crud.EnumControl
	Pattern        string
	PatternMessage crud.MessageCode
	Lookup         *crud.LookupDefinition
}

// AutoTenantTableOption customizes one conventional tenant-scoped table.
// Options are evaluated only during application bootstrap.
type AutoTenantTableOption func(*AutoTableConfig) error

// WithMaxLength sets the maximum accepted character count for one text field.
// The field key and bound are server configuration. Bounds must remain finite
// to protect the HTTP, validation, and persistence layers from oversized input.
func WithMaxLength(field crud.FieldKey, maxLength int) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(field)) {
			return fmt.Errorf("sqladapter: invalid field override key %q", field)
		}
		if maxLength < 1 || maxLength > maximumAutoTextMaxLength {
			return fmt.Errorf("sqladapter: text maximum must be between 1 and %d", maximumAutoTextMaxLength)
		}
		if config.FieldOverrides == nil {
			config.FieldOverrides = make(map[crud.FieldKey]FieldOverride)
		}
		override := config.FieldOverrides[field]
		override.MaxLength = maxLength
		config.FieldOverrides[field] = override
		return nil
	}
}

// WithPattern declares a server-owned RE2 validator for one string-backed
// automatic field. The message code is returned when the value does not match.
func WithPattern(field crud.FieldKey, pattern string, message crud.MessageCode) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(field)) {
			return fmt.Errorf("sqladapter: invalid field override key %q", field)
		}
		if pattern == "" {
			return fmt.Errorf("sqladapter: pattern for field %q must not be empty", field)
		}
		if len(pattern) > maximumAutoPatternLength {
			return fmt.Errorf("sqladapter: pattern for field %q must not exceed %d characters", field, maximumAutoPatternLength)
		}
		if message == "" {
			return fmt.Errorf("sqladapter: pattern message for field %q is required", field)
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("sqladapter: invalid pattern for field %q: %v", field, err)
		}
		if config.FieldOverrides == nil {
			config.FieldOverrides = make(map[crud.FieldKey]FieldOverride)
		}
		override := config.FieldOverrides[field]
		override.Pattern = pattern
		override.PatternMessage = message
		config.FieldOverrides[field] = override
		return nil
	}
}

// WithOptional allows an automatic field to be empty. The field remains
// required unless a consumer explicitly declares this exception at bootstrap.
func WithOptional(field crud.FieldKey) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(field)) {
			return fmt.Errorf("sqladapter: invalid field override key %q", field)
		}
		if config.FieldOverrides == nil {
			config.FieldOverrides = make(map[crud.FieldKey]FieldOverride)
		}
		override := config.FieldOverrides[field]
		override.Optional = true
		config.FieldOverrides[field] = override
		return nil
	}
}

// WithDefault declares a static create-time default for one automatic field.
// The value is shown by the renderer and is also applied by clear.crud when a
// create mutation omits the field. Dynamic defaults remain adapter-owned.
func WithDefault(field crud.FieldKey, value crud.Value) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(field)) {
			return fmt.Errorf("sqladapter: invalid field override key %q", field)
		}
		if !autoDefaultValue(value) {
			return fmt.Errorf("sqladapter: default for field %q must be a non-null scalar", field)
		}
		if config.FieldOverrides == nil {
			config.FieldOverrides = make(map[crud.FieldKey]FieldOverride)
		}
		override := config.FieldOverrides[field]
		override.Default = value
		config.FieldOverrides[field] = override
		return nil
	}
}

// WithBooleanDisplay chooses the symbols rendered for one boolean field. The
// choice is explicit consumer configuration and is carried by the generic
// definition; it is never inferred from a database column name.
func WithBooleanDisplay(field crud.FieldKey, trueSymbol, falseSymbol string) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(field)) {
			return fmt.Errorf("sqladapter: invalid field override key %q", field)
		}
		if trueSymbol == "" || falseSymbol == "" {
			return fmt.Errorf("sqladapter: boolean display requires true and false symbols")
		}
		if config.FieldOverrides == nil {
			config.FieldOverrides = make(map[crud.FieldKey]FieldOverride)
		}
		override := config.FieldOverrides[field]
		override.BooleanDisplay = &crud.BooleanDisplay{True: trueSymbol, False: falseSymbol}
		config.FieldOverrides[field] = override
		return nil
	}
}

// WithEnum promotes one automatically discovered scalar field to an enum.
// The allowed values and their labels are server-owned consumer configuration;
// they are never inferred from a column name or accepted from HTTP input.
func WithEnum(field crud.FieldKey, options ...crud.Option) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(field)) {
			return fmt.Errorf("sqladapter: invalid field override key %q", field)
		}
		if len(options) == 0 {
			return fmt.Errorf("sqladapter: enum %q requires at least one option", field)
		}
		copyOptions := append([]crud.Option(nil), options...)
		seen := make(map[string]struct{}, len(copyOptions))
		for index, option := range copyOptions {
			if option.Label == "" {
				return fmt.Errorf("sqladapter: enum %q option %d requires a label", field, index)
			}
			identity := fmt.Sprintf("%T:%v", option.Value, option.Value)
			if _, exists := seen[identity]; exists {
				return fmt.Errorf("sqladapter: enum %q contains duplicate values", field)
			}
			seen[identity] = struct{}{}
		}
		if config.FieldOverrides == nil {
			config.FieldOverrides = make(map[crud.FieldKey]FieldOverride)
		}
		override := config.FieldOverrides[field]
		override.Enum = copyOptions
		config.FieldOverrides[field] = override
		return nil
	}
}

// WithEnumControl selects the desktop control for one enum field. Mobile
// renderers may still use a native select when the viewport is constrained.
func WithEnumControl(field crud.FieldKey, control crud.EnumControl) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(field)) {
			return fmt.Errorf("sqladapter: invalid field override key %q", field)
		}
		switch control {
		case crud.EnumControlAuto, crud.EnumControlSelect, crud.EnumControlRadio, crud.EnumControlSegmented, crud.EnumControlButtons:
		default:
			return fmt.Errorf("sqladapter: invalid enum control %q", control)
		}
		if config.FieldOverrides == nil {
			config.FieldOverrides = make(map[crud.FieldKey]FieldOverride)
		}
		override := config.FieldOverrides[field]
		override.EnumControl = control
		config.FieldOverrides[field] = override
		return nil
	}
}

// WithLookup declares a server-owned dependent lookup for one automatic
// field. The target resource and dependency graph are registered metadata;
// they are never selected by HTTP input.
func WithLookup(field crud.FieldKey, lookup crud.LookupDefinition) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(field)) || !autoKey(string(lookup.Resource)) || !autoKey(string(lookup.ValueField)) || !autoKey(string(lookup.LabelField)) {
			return fmt.Errorf("sqladapter: invalid lookup mapping for %q", field)
		}
		if lookup.PageSize == 0 || lookup.PageSize > 100 {
			return fmt.Errorf("sqladapter: lookup page size must be between 1 and 100")
		}
		seen := make(map[crud.FieldKey]struct{}, len(lookup.Dependencies))
		for _, dependency := range lookup.Dependencies {
			if !autoKey(string(dependency)) {
				return fmt.Errorf("sqladapter: invalid lookup dependency %q", dependency)
			}
			if _, exists := seen[dependency]; exists {
				return fmt.Errorf("sqladapter: duplicate lookup dependency %q", dependency)
			}
			seen[dependency] = struct{}{}
		}
		seenFilters := make(map[crud.FieldKey]struct{}, len(lookup.FixedFilters))
		for _, filter := range lookup.FixedFilters {
			if !autoKey(string(filter.Field)) || len(filter.Values) == 0 || len(filter.Values) > 100 {
				return fmt.Errorf("sqladapter: invalid fixed lookup filter for %q", field)
			}
			for _, value := range filter.Values {
				if !autoLookupValue(value) {
					return fmt.Errorf("sqladapter: invalid fixed lookup filter for %q", field)
				}
			}
			if _, exists := seenFilters[filter.Field]; exists {
				return fmt.Errorf("sqladapter: duplicate fixed lookup filter %q", filter.Field)
			}
			seenFilters[filter.Field] = struct{}{}
		}
		if config.FieldOverrides == nil {
			config.FieldOverrides = make(map[crud.FieldKey]FieldOverride)
		}
		copyLookup := lookup
		copyLookup.Dependencies = append([]crud.FieldKey(nil), lookup.Dependencies...)
		copyLookup.FixedFilters = cloneAutoLookupFilters(lookup.FixedFilters)
		override := config.FieldOverrides[field]
		override.Lookup = &copyLookup
		config.FieldOverrides[field] = override
		return nil
	}
}

func autoLookupValue(value crud.Value) bool {
	switch value.(type) {
	case string, bool, int64:
		return true
	default:
		return false
	}
}

func autoDefaultValue(value crud.Value) bool {
	switch value.(type) {
	case string, bool, int64:
		return true
	default:
		return false
	}
}

func cloneAutoLookupFilters(filters []crud.FixedLookupFilter) []crud.FixedLookupFilter {
	clone := make([]crud.FixedLookupFilter, len(filters))
	for index, filter := range filters {
		clone[index] = filter
		clone[index].Values = append([]crud.Value(nil), filter.Values...)
	}
	return clone
}

// WithResourceKey registers another server-owned view over the same physical
// table. This is useful when one storage resource needs a focused projection
// or presentation variant without duplicating persistence or HTTP handlers.
func WithResourceKey(key crud.ResourceKey) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if !autoKey(string(key)) {
			return fmt.Errorf("sqladapter: invalid resource key %q", key)
		}
		config.Key = key
		return nil
	}
}

// WithGridColumns explicitly selects the fields shown in the grid. All
// generated fields remain available to the form; this option only changes the
// server-owned grid projection.
func WithGridColumns(columns ...crud.FieldKey) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if len(columns) == 0 {
			return fmt.Errorf("sqladapter: grid columns require at least one field")
		}
		seen := make(map[crud.FieldKey]struct{}, len(columns))
		for _, column := range columns {
			if !autoKey(string(column)) {
				return fmt.Errorf("sqladapter: invalid grid column %q", column)
			}
			if _, exists := seen[column]; exists {
				return fmt.Errorf("sqladapter: duplicate grid column %q", column)
			}
			seen[column] = struct{}{}
		}
		config.GridColumns = append([]crud.FieldKey(nil), columns...)
		return nil
	}
}

// WithFormFields explicitly selects and orders fields shown by the form.
// Empty configuration keeps the default of all visible fields in definition
// order.
func WithFormFields(fields ...crud.FieldKey) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if len(fields) == 0 {
			return fmt.Errorf("sqladapter: form fields require at least one field")
		}
		seen := make(map[crud.FieldKey]struct{}, len(fields))
		for _, field := range fields {
			if !autoKey(string(field)) {
				return fmt.Errorf("sqladapter: invalid form field %q", field)
			}
			if _, exists := seen[field]; exists {
				return fmt.Errorf("sqladapter: duplicate form field %q", field)
			}
			seen[field] = struct{}{}
		}
		config.FormFields = append([]crud.FieldKey(nil), fields...)
		return nil
	}
}

// WithGridPageSize selects the default number of rows requested by the grid.
// The value is bounded and added to the standard allowed sizes. This is a
// server-owned presentation choice; it does not change the adapter's maximum
// page size or permit an unbounded query.
func WithGridPageSize(size uint16) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if size == 0 || size > 100 {
			return fmt.Errorf("sqladapter: grid page size must be between 1 and 100")
		}
		config.GridPageSize = size
		return nil
	}
}

// WithDefaultSort declares an explicit server-owned ordering for the automatic
// collection. The SQLite schema must provide an index compatible with all
// terms; registration fails during bootstrap when it does not.
func WithDefaultSort(terms ...crud.Sort) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if len(terms) == 0 {
			return fmt.Errorf("sqladapter: default sort requires at least one field")
		}
		config.DefaultSort = append([]crud.Sort(nil), terms...)
		return nil
	}
}

// WithSoftDelete declares the server-owned column used for soft deletion.
// Soft-deleted records are excluded from normal reads by the data adapter.
func WithSoftDelete(column Identifier) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if config.DeleteMode == crud.DeleteModeHardDelete {
			return fmt.Errorf("sqladapter: soft-delete and hard-delete are mutually exclusive")
		}
		if !validIdentifier(column) {
			return fmt.Errorf("sqladapter: invalid soft-delete column")
		}
		declared := column
		config.ArchiveColumn = &declared
		config.DeleteMode = crud.DeleteModeArchive
		return nil
	}
}

// WithArchiveVisibility allows a grid to include archived records in its
// server-owned list query. Active records remain the default; archived rows
// are returned as read-only metadata and are never made editable by this
// option.
func WithArchiveVisibility(visibility crud.ArchiveVisibility) AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if visibility != crud.ArchiveVisibilityActiveOnly && visibility != crud.ArchiveVisibilityActiveAndArchived {
			return fmt.Errorf("sqladapter: unknown archive visibility %q", visibility)
		}
		config.ArchiveVisibility = visibility
		return nil
	}
}

// WithHardDelete explicitly enables physical deletion for a conventional
// tenant-scoped table. Without this opt-in, automatic tables expose no delete
// action; the engine never guesses that a DELETE is safe.
func WithHardDelete() AutoTenantTableOption {
	return func(config *AutoTableConfig) error {
		if config.ArchiveColumn != nil || config.DeleteMode == crud.DeleteModeArchive {
			return fmt.Errorf("sqladapter: hard-delete and soft-delete are mutually exclusive")
		}
		config.DeleteMode = crud.DeleteModeHardDelete
		return nil
	}
}

// AutoTableConfig supplies the server-owned choices which schema inspection
// must never infer: the resource key, trusted scope, and authorization gates.
// All other conventional details are derived from the SQLite table metadata.
type AutoTableConfig struct {
	Table             Identifier
	Key               crud.ResourceKey
	ScopeColumns      map[string]Identifier
	Permissions       crud.Permissions
	FieldOverrides    map[crud.FieldKey]FieldOverride
	GridColumns       []crud.FieldKey
	FormFields        []crud.FieldKey
	DefaultSort       []crud.Sort
	GridPageSize      uint16
	ArchiveColumn     *Identifier
	ArchiveVisibility crud.ArchiveVisibility
	Global            bool
	DeleteMode        crud.DeleteMode
}

// AutoTenantTable applies the explicit conventional tenant mapping: a trusted
// tenant_id scope and standard permission names derived from the resource key.
// The host still resolves and authorizes every permission for each request. If
// no archive option is supplied, the generated definition has no delete action.
func AutoTenantTable(ctx context.Context, db *sql.DB, table Identifier, options ...AutoTenantTableOption) (crud.Definition, error) {
	key := crud.ResourceKey(table)
	defaultPermissions := autoPermissions(key)
	config := AutoTableConfig{
		Table:        table,
		Key:          key,
		ScopeColumns: map[string]Identifier{"tenant_id": "tenant_id"},
		Permissions:  defaultPermissions,
	}
	for _, option := range options {
		if option == nil {
			return crud.Definition{}, fmt.Errorf("sqladapter: nil auto table option")
		}
		if err := option(&config); err != nil {
			return crud.Definition{}, err
		}
	}
	if config.Key != key && config.Permissions == defaultPermissions {
		config.Permissions = autoPermissions(config.Key)
	}
	// A table without an explicitly declared archive policy has no delete
	// action. Do not make the table-only convention fail because the generated
	// permission set includes a delete gate that the source cannot implement.
	if config.ArchiveColumn == nil && config.DeleteMode == "" && config.Permissions.Delete == autoPermissions(config.Key).Delete {
		config.Permissions.Delete = ""
	}
	return AutoTable(ctx, db, config)
}

// AutoGlobalTable creates an explicitly global, read-only reference resource.
// It has no tenant column and cannot be mutated through clear.crud. Use it for
// stable catalogs such as postal geography, vehicle makes, or airport data.
func AutoGlobalTable(ctx context.Context, db *sql.DB, table Identifier, options ...AutoTenantTableOption) (crud.Definition, error) {
	key := crud.ResourceKey(table)
	config := AutoTableConfig{Table: table, Key: key, Global: true, Permissions: crud.Permissions{Read: "crud." + string(key) + ".read"}}
	for _, option := range options {
		if option == nil {
			return crud.Definition{}, fmt.Errorf("sqladapter: nil auto table option")
		}
		if err := option(&config); err != nil {
			return crud.Definition{}, err
		}
	}
	config.Permissions = crud.Permissions{Read: "crud." + string(config.Key) + ".read"}
	return AutoTable(ctx, db, config)
}

func autoPermissions(key crud.ResourceKey) crud.Permissions {
	return crud.Permissions{
		Read: "crud." + string(key) + ".read", Create: "crud." + string(key) + ".create",
		Update: "crud." + string(key) + ".update", Delete: "crud." + string(key) + ".delete",
	}
}

// RegisterAutoTenantTable creates and registers one conventional tenant-scoped
// resource. The table is server configuration, never an HTTP value.
func RegisterAutoTenantTable(ctx context.Context, db *sql.DB, registry *crud.Registry, table Identifier, options ...AutoTenantTableOption) error {
	definition, err := AutoTenantTable(ctx, db, table, options...)
	if err != nil {
		return err
	}
	return registry.Register(ctx, definition)
}

// RegisterAutoGlobalTable registers one explicit global, read-only reference
// resource in the server-owned registry.
func RegisterAutoGlobalTable(ctx context.Context, db *sql.DB, registry *crud.Registry, table Identifier, options ...AutoTenantTableOption) error {
	definition, err := AutoGlobalTable(ctx, db, table, options...)
	if err != nil {
		return err
	}
	return registry.Register(ctx, definition)
}

// AutoTable creates a conventional CRUD definition from an existing SQLite
// table. It performs inspection only during application bootstrap; requests
// use the resulting immutable SimpleTable and never inspect a schema.
//
// The table and scope mappings are server configuration. AutoTable does not
// infer a tenant boundary, permissions, sensitivity, or domain meaning from a
// table or column name. A supported table has an integer primary key named id,
// an integer version column, at least one non-technical scalar column, and the
// configured trusted scope columns. A configured integer or boolean soft-delete
// column enables the archive deletion policy.
func AutoTable(ctx context.Context, db *sql.DB, config AutoTableConfig) (crud.Definition, error) {
	if db == nil {
		return crud.Definition{}, fmt.Errorf("sqladapter: nil database")
	}
	if !validIdentifier(config.Table) {
		return crud.Definition{}, fmt.Errorf("sqladapter: invalid table")
	}
	if config.Key == "" {
		config.Key = crud.ResourceKey(config.Table)
	}
	if !autoKey(string(config.Key)) {
		return crud.Definition{}, fmt.Errorf("sqladapter: invalid resource key")
	}
	if !config.Global && len(config.ScopeColumns) == 0 {
		return crud.Definition{}, fmt.Errorf("sqladapter: trusted scope columns are required")
	}
	if config.Global && len(config.ScopeColumns) != 0 {
		return crud.Definition{}, fmt.Errorf("sqladapter: global table cannot declare trusted scope columns")
	}
	if config.Permissions.Read == "" {
		return crud.Definition{}, fmt.Errorf("sqladapter: read permission is required")
	}

	columns, err := inspectTable(ctx, db, config.Table)
	if err != nil {
		return crud.Definition{}, err
	}
	return autoDefinition(ctx, db, config, columns)
}

type sqliteColumn struct {
	name        Identifier
	declared    string
	ordinal     int
	notNull     bool
	primaryKey  bool
	defaultExpr string
}

func inspectTable(ctx context.Context, db *sql.DB, table Identifier) (map[Identifier]sqliteColumn, error) {
	// Identifier is validated before interpolation. SQLite does not accept a
	// parameter for a PRAGMA object name, so its value cannot come from HTTP.
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+quote(table)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[Identifier]sqliteColumn)
	for rows.Next() {
		var index, notNull, primaryKey int
		var name, declared string
		var defaultValue sql.NullString
		if err := rows.Scan(&index, &name, &declared, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		identifier, err := NewIdentifier(name)
		if err != nil {
			return nil, fmt.Errorf("sqladapter: table %q has unsupported column %q", table, name)
		}
		columns[identifier] = sqliteColumn{name: identifier, declared: declared, ordinal: index, notNull: notNull != 0, primaryKey: primaryKey != 0, defaultExpr: defaultValue.String}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("sqladapter: table %q does not exist or has no columns", table)
	}
	return columns, nil
}

func autoDefinition(ctx context.Context, db *sql.DB, config AutoTableConfig, columns map[Identifier]sqliteColumn) (crud.Definition, error) {
	id := Identifier("id")
	version := Identifier("version")
	if column, ok := columns[id]; !ok || !column.primaryKey || sqliteType(column.declared) != crud.FieldInteger {
		return crud.Definition{}, fmt.Errorf("sqladapter: table %q requires integer primary key id", config.Table)
	}
	if column, ok := columns[version]; !ok || sqliteType(column.declared) != crud.FieldInteger {
		return crud.Definition{}, fmt.Errorf("sqladapter: table %q requires integer version", config.Table)
	}
	for scopeKey, column := range config.ScopeColumns {
		if !autoKey(scopeKey) || !validIdentifier(column) {
			return crud.Definition{}, fmt.Errorf("sqladapter: invalid trusted scope mapping")
		}
		if _, ok := columns[column]; !ok {
			return crud.Definition{}, fmt.Errorf("sqladapter: scope column %q is absent from table %q", column, config.Table)
		}
	}

	technical := map[Identifier]struct{}{id: {}, version: {}}
	for _, column := range config.ScopeColumns {
		technical[column] = struct{}{}
	}
	table := TableDefinition{Table: config.Table, IDColumn: id, VersionColumn: version, ScopeColumns: config.ScopeColumns, Global: config.Global, Fields: make(map[crud.FieldKey]Identifier)}
	if config.ArchiveColumn != nil {
		column, ok := columns[*config.ArchiveColumn]
		if !ok {
			return crud.Definition{}, fmt.Errorf("sqladapter: soft-delete column %q is absent from table %q", *config.ArchiveColumn, config.Table)
		}
		fieldType := sqliteType(column.declared)
		if fieldType != crud.FieldInteger && fieldType != crud.FieldBoolean {
			return crud.Definition{}, fmt.Errorf("sqladapter: soft-delete column %q must be integer or boolean", *config.ArchiveColumn)
		}
		archive := *config.ArchiveColumn
		table.ArchiveColumn = &archive
		technical[archive] = struct{}{}
	}
	archiveVisibility := config.ArchiveVisibility
	if archiveVisibility == "" {
		archiveVisibility = crud.ArchiveVisibilityActiveOnly
	}
	if archiveVisibility != crud.ArchiveVisibilityActiveOnly && archiveVisibility != crud.ArchiveVisibilityActiveAndArchived {
		return crud.Definition{}, fmt.Errorf("sqladapter: unknown archive visibility %q", archiveVisibility)
	}
	if archiveVisibility == crud.ArchiveVisibilityActiveAndArchived && table.ArchiveColumn == nil {
		return crud.Definition{}, fmt.Errorf("sqladapter: archive visibility requires a soft-delete column")
	}

	fields, physicalOrder, err := autoFields(columns, technical, table.Fields, config.FieldOverrides)
	if err != nil {
		return crud.Definition{}, err
	}
	// Definition.Fields and the automatic grid preserve physical schema order.
	// Explicit consumer projections remain authoritative. Record ordering is a
	// separate concern and keeps its deterministic fallback below.
	ordered := append([]crud.FieldKey(nil), physicalOrder...)
	source, err := NewSimpleTable(db, table)
	if err != nil {
		return crud.Definition{}, err
	}
	metadata, err := source.Metadata(ctx)
	if err != nil {
		return crud.Definition{}, err
	}
	if err := applyAutoSchemaEnums(fields, metadata.Fields, config.FieldOverrides); err != nil {
		return crud.Definition{}, err
	}
	gridColumns, err := autoGridColumns(ordered, config.GridColumns)
	if err != nil {
		return crud.Definition{}, err
	}
	sortOrder := append([]crud.FieldKey(nil), ordered...)
	sort.Slice(sortOrder, func(left, right int) bool { return sortOrder[left] < sortOrder[right] })
	sortColumns := append([]crud.FieldKey(nil), gridColumns...)
	if len(config.GridColumns) == 0 {
		sort.Slice(sortColumns, func(left, right int) bool { return sortColumns[left] < sortColumns[right] })
	}
	if err := validateAutoFieldSelection("form", config.FormFields, fields); err != nil {
		return crud.Definition{}, err
	}
	pageSize, pageSizes, err := autoGridPagination(config.GridPageSize)
	if err != nil {
		return crud.Definition{}, err
	}
	indexes, err := inspectIndexes(ctx, db, config.Table)
	if err != nil {
		return crud.Definition{}, err
	}
	defaultSort, err := autoDefaultSort(config, table, fields, sortOrder, sortColumns, indexes)
	if err != nil {
		return crud.Definition{}, err
	}
	searchable := make([]crud.FieldKey, 0, len(ordered))
	for _, field := range fields {
		if field.Type == crud.FieldString || field.Type == crud.FieldText || field.Type == crud.FieldEnum {
			searchable = append(searchable, field.Key)
		}
	}
	deleteMode := config.DeleteMode
	if deleteMode == "" {
		deleteMode = crud.DeleteModeNone
	}
	permissions := config.Permissions
	if table.ArchiveColumn != nil {
		if deleteMode != crud.DeleteModeNone && deleteMode != crud.DeleteModeArchive {
			return crud.Definition{}, fmt.Errorf("sqladapter: archive column conflicts with delete mode")
		}
		deleteMode = crud.DeleteModeArchive
		if permissions.Delete == "" {
			return crud.Definition{}, fmt.Errorf("sqladapter: archive table requires delete permission")
		}
	} else if deleteMode == crud.DeleteModeHardDelete {
		if permissions.Delete == "" {
			return crud.Definition{}, fmt.Errorf("sqladapter: hard-delete requires delete permission")
		}
	} else if permissions.Delete != "" {
		return crud.Definition{}, fmt.Errorf("sqladapter: delete permission requires archived column")
	}

	return crud.Definition{
		Contract: crud.ContractDefinitionV1,
		Key:      config.Key,
		Labels:   crud.Labels{Title: message(config.Key, "title"), Singular: message(config.Key, "singular")},
		Scope:    scopeRequirements(config.ScopeColumns, config.Global), Permissions: permissions,
		Fields:       fields,
		Grid:         crud.GridDefinition{Columns: gridColumns, Searchable: searchable, Sortable: sortOrder, DefaultSort: defaultSort, Pagination: crud.PaginationDefinition{Mode: crud.PageModeOffset, DefaultSize: pageSize, AllowedSizes: pageSizes, Total: true}, ArchiveVisibility: archiveVisibility},
		Form:         crud.FormDefinition{Fields: append([]crud.FieldKey(nil), config.FormFields...)},
		Presentation: crud.Presentation{Collection: crud.CollectionAuto, Density: crud.DensityCompact},
		Source:       source, UOW: source, Delete: crud.DeletePolicy{Mode: deleteMode}, Concurrency: crud.ConcurrencyPolicy{Mode: crud.ConcurrencyVersion},
	}, nil
}

func applyAutoSchemaEnums(fields []crud.Field, metadata map[crud.FieldKey]crud.FieldMetadata, overrides map[crud.FieldKey]FieldOverride) error {
	for index := range fields {
		field := &fields[index]
		facts, ok := metadata[field.Key]
		if !ok || len(facts.Enum) == 0 {
			continue
		}
		if field.Type == crud.FieldBoolean {
			continue
		}
		if field.Lookup != nil {
			// A consumer-declared lookup is more specific than an enum
			// inferred from a schema CHECK constraint.
			continue
		}
		override := overrides[field.Key]
		if field.Type == crud.FieldEnum {
			for optionIndex, option := range field.Enum {
				if !containsSchemaEnumValue(facts.Enum, option.Value) {
					return fmt.Errorf("sqladapter: enum override %q option %d is not allowed by the schema", field.Key, optionIndex)
				}
			}
			continue
		}
		field.Type = crud.FieldEnum
		field.Enum = make([]crud.Option, 0, len(facts.Enum))
		for _, value := range facts.Enum {
			field.Enum = append(field.Enum, crud.Option{Value: value, Label: crud.MessageCode(schemaEnumLabel(value))})
		}
		if override.EnumControl != "" {
			field.EnumControl = override.EnumControl
		}
	}
	return nil
}

func containsSchemaEnumValue(values []crud.Value, wanted crud.Value) bool {
	wantedIdentity := fmt.Sprintf("%T:%v", wanted, wanted)
	for _, value := range values {
		if fmt.Sprintf("%T:%v", value, value) == wantedIdentity {
			return true
		}
	}
	return false
}

func schemaEnumLabel(value crud.Value) string {
	switch typed := value.(type) {
	case nil:
		return "∅"
	case string:
		if typed == "" {
			return "∅"
		}
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprint(typed)
	}
}

func autoGridPagination(configured uint16) (uint16, []uint16, error) {
	pageSize := uint16(defaultAutoGridPageSize)
	if configured != 0 {
		pageSize = configured
	}
	if pageSize > 100 {
		return 0, nil, fmt.Errorf("sqladapter: grid page size must be between 1 and 100")
	}
	pageSizes := []uint16{25, 50, 100}
	if configured != 0 {
		pageSizes = append(pageSizes, configured)
	}
	sort.Slice(pageSizes, func(left, right int) bool { return pageSizes[left] < pageSizes[right] })
	unique := pageSizes[:0]
	for _, size := range pageSizes {
		if len(unique) == 0 || unique[len(unique)-1] != size {
			unique = append(unique, size)
		}
	}
	return pageSize, append([]uint16(nil), unique...), nil
}

func autoGridColumns(ordered, configured []crud.FieldKey) ([]crud.FieldKey, error) {
	if len(configured) == 0 {
		limit := defaultAutoGridColumns
		if len(ordered) < limit {
			limit = len(ordered)
		}
		return append([]crud.FieldKey(nil), ordered[:limit]...), nil
	}
	known := make(map[crud.FieldKey]struct{}, len(ordered))
	for _, key := range ordered {
		known[key] = struct{}{}
	}
	for _, key := range configured {
		if _, ok := known[key]; !ok {
			return nil, fmt.Errorf("sqladapter: grid column %q does not match a conventional field", key)
		}
	}
	return append([]crud.FieldKey(nil), configured...), nil
}

func validateAutoFieldSelection(kind string, configured []crud.FieldKey, fields []crud.Field) error {
	if len(configured) == 0 {
		return nil
	}
	known := make(map[crud.FieldKey]crud.Field, len(fields))
	for _, field := range fields {
		known[field.Key] = field
	}
	for _, key := range configured {
		field, ok := known[key]
		if !ok || !field.Visible {
			return fmt.Errorf("sqladapter: %s field %q does not match a visible conventional field", kind, key)
		}
	}
	return nil
}

func autoFields(columns map[Identifier]sqliteColumn, technical map[Identifier]struct{}, mapping map[crud.FieldKey]Identifier, overrides map[crud.FieldKey]FieldOverride) ([]crud.Field, []crud.FieldKey, error) {
	names := make([]string, 0, len(columns))
	for column := range columns {
		if _, skip := technical[column]; !skip {
			names = append(names, string(column))
		}
	}
	sort.Slice(names, func(left, right int) bool {
		leftColumn := columns[Identifier(names[left])]
		rightColumn := columns[Identifier(names[right])]
		if leftColumn.ordinal != rightColumn.ordinal {
			return leftColumn.ordinal < rightColumn.ordinal
		}
		return names[left] < names[right]
	})
	fields := make([]crud.Field, 0, len(names))
	keys := make([]crud.FieldKey, 0, len(names))
	usedOverrides := make(map[crud.FieldKey]struct{}, len(overrides))
	for _, name := range names {
		column := columns[Identifier(name)]
		fieldType := sqliteType(column.declared)
		if fieldType == "" {
			return nil, nil, fmt.Errorf("sqladapter: column %q has unsupported SQLite type %q", column.name, column.declared)
		}
		key := crud.FieldKey(column.name)
		mapping[key] = column.name
		field := crud.Field{Key: key, Label: crud.MessageCode("crud.field." + name), Type: fieldType, Visible: true}
		override, overridden := overrides[key]
		if column.defaultExpr != "" {
			if value, ok := sqliteStaticDefault(column.defaultExpr, fieldType); ok {
				field.Default = value
			}
		}
		if overridden && override.Optional {
			if column.notNull {
				return nil, nil, fmt.Errorf("sqladapter: optional field override %q requires a nullable column", key)
			}
			field.Optional = true
			usedOverrides[key] = struct{}{}
		}
		if fieldType == crud.FieldString || fieldType == crud.FieldText {
			field.MaxLength = defaultTextMaxLength(column.declared)
			if overridden && override.MaxLength != 0 {
				if override.MaxLength < 1 || override.MaxLength > maximumAutoTextMaxLength {
					return nil, nil, fmt.Errorf("sqladapter: text maximum for field %q must be between 1 and %d", key, maximumAutoTextMaxLength)
				}
				field.MaxLength = override.MaxLength
				usedOverrides[key] = struct{}{}
			}
		} else if overridden && override.MaxLength != 0 {
			return nil, nil, fmt.Errorf("sqladapter: field override %q requires a text field", key)
		}
		if overridden && override.Lookup != nil && len(override.Enum) != 0 {
			return nil, nil, fmt.Errorf("sqladapter: field override %q cannot be both enum and lookup", key)
		}
		if overridden && override.Lookup != nil {
			field.Type = crud.FieldLookup
			lookup := *override.Lookup
			lookup.Dependencies = append([]crud.FieldKey(nil), override.Lookup.Dependencies...)
			field.Lookup = &lookup
			usedOverrides[key] = struct{}{}
		}
		if overridden && len(override.Enum) != 0 {
			if fieldType != crud.FieldString && fieldType != crud.FieldText {
				return nil, nil, fmt.Errorf("sqladapter: enum override %q requires a text field", key)
			}
			field.Type = crud.FieldEnum
			field.Enum = append([]crud.Option(nil), override.Enum...)
			usedOverrides[key] = struct{}{}
		}
		if overridden && override.EnumControl != "" {
			if field.Type != crud.FieldEnum {
				return nil, nil, fmt.Errorf("sqladapter: enum control override %q requires an enum field", key)
			}
			field.EnumControl = override.EnumControl
			usedOverrides[key] = struct{}{}
		}
		if overridden && override.BooleanDisplay != nil {
			if fieldType != crud.FieldBoolean {
				return nil, nil, fmt.Errorf("sqladapter: boolean display override %q requires a boolean field", key)
			}
			booleanDisplay := *override.BooleanDisplay
			field.BooleanDisplay = &booleanDisplay
			usedOverrides[key] = struct{}{}
		}
		if overridden && override.Pattern != "" {
			if !autoPatternFieldType(field.Type) {
				return nil, nil, fmt.Errorf("sqladapter: pattern override %q requires a string-backed field", key)
			}
			field.Pattern = override.Pattern
			field.PatternMessage = override.PatternMessage
			usedOverrides[key] = struct{}{}
		}
		if overridden && override.Default != nil {
			field.Default = override.Default
			usedOverrides[key] = struct{}{}
		}
		fields, keys = append(fields, field), append(keys, key)
	}
	if len(fields) == 0 {
		return nil, nil, fmt.Errorf("sqladapter: table has no conventional fields")
	}
	for key := range overrides {
		if _, ok := usedOverrides[key]; !ok {
			return nil, nil, fmt.Errorf("sqladapter: field override %q does not match a conventional field", key)
		}
	}
	return fields, keys, nil
}

func sqliteType(declared string) crud.FieldType {
	typeName := strings.ToUpper(strings.TrimSpace(declared))
	switch {
	case strings.Contains(typeName, "BOOL"):
		return crud.FieldBoolean
	case strings.Contains(typeName, "INT"):
		return crud.FieldInteger
	case strings.Contains(typeName, "DEC"), strings.Contains(typeName, "NUM"), strings.Contains(typeName, "REAL"), strings.Contains(typeName, "FLOA"), strings.Contains(typeName, "DOUB"):
		return crud.FieldDecimal
	case strings.Contains(typeName, "DATETIME"), strings.Contains(typeName, "TIMESTAMP"):
		return crud.FieldDateTime
	case strings.Contains(typeName, "DATE"):
		return crud.FieldDate
	case strings.Contains(typeName, "CHAR"), strings.Contains(typeName, "CLOB"), strings.Contains(typeName, "TEXT"), typeName == "":
		return crud.FieldString
	default:
		return ""
	}
}

func autoPatternFieldType(fieldType crud.FieldType) bool {
	switch fieldType {
	case crud.FieldString, crud.FieldText, crud.FieldDecimal, crud.FieldDate, crud.FieldDateTime, crud.FieldEmail, crud.FieldPhone:
		return true
	default:
		return false
	}
}

func varcharLength(declared string) int {
	start, end := strings.IndexByte(declared, '('), strings.IndexByte(declared, ')')
	if start < 0 || end <= start+1 {
		return 0
	}
	length, err := strconv.Atoi(strings.TrimSpace(declared[start+1 : end]))
	if err != nil || length < 1 {
		return 0
	}
	return length
}

func defaultTextMaxLength(declared string) int {
	declaredLength := varcharLength(declared)
	if declaredLength > 0 && declaredLength < defaultAutoTextMaxLength {
		return declaredLength
	}
	return defaultAutoTextMaxLength
}

func scopeRequirements(scope map[string]Identifier, global bool) crud.ScopeRequirements {
	keys := make([]string, 0, len(scope))
	for key := range scope {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	mode := crud.ScopeModeTenant
	if global {
		mode = crud.ScopeModeGlobal
	}
	return crud.ScopeRequirements{Mode: mode, Keys: keys}
}

func message(key crud.ResourceKey, suffix string) crud.MessageCode {
	return crud.MessageCode("crud." + string(key) + "." + suffix)
}

func autoKey(value string) bool {
	return identifierPattern.MatchString(value) && value == strings.ToLower(value)
}
