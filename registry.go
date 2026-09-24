package crud

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"sync"
)

var (
	// ErrRegistrySealed reports an attempt to register after startup is complete.
	ErrRegistrySealed = errors.New("crud registry is sealed")
	// ErrDuplicateResource reports an attempt to replace a registered resource.
	ErrDuplicateResource = errors.New("crud resource is already registered")
)

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// DefinitionError reports an invalid server-side definition. It is intended for
// startup diagnostics and must never cross an operator-facing transport.
type DefinitionError struct {
	Path   string
	Reason string
}

// Error implements error.
func (err *DefinitionError) Error() string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("invalid CRUD definition at %s: %s", err.Path, err.Reason)
}

// Registry stores definitions registered during application startup.
// Definitions are copied on registration and retrieval so callers cannot mutate
// the registry after validation.
type Registry struct {
	mu          sync.RWMutex
	definitions map[ResourceKey]Definition
	sealed      bool
}

// NewRegistry creates an empty mutable registry.
func NewRegistry() *Registry {
	return &Registry{definitions: make(map[ResourceKey]Definition)}
}

// Register validates and stores one definition. Registration is rejected after
// Seal and duplicate keys never replace an existing resource.
func (registry *Registry) Register(ctx context.Context, definition Definition) error {
	if registry == nil {
		return errors.New("crud registry is nil")
	}
	if err := ValidateDefinition(ctx, definition); err != nil {
		return err
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if registry.sealed {
		return ErrRegistrySealed
	}
	if _, exists := registry.definitions[definition.Key]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateResource, definition.Key)
	}
	if registry.definitions == nil {
		registry.definitions = make(map[ResourceKey]Definition)
	}
	registry.definitions[definition.Key] = cloneDefinition(definition)
	return nil
}

// Get returns a defensive copy of a registered definition.
func (registry *Registry) Get(key ResourceKey) (Definition, bool) {
	if registry == nil {
		return Definition{}, false
	}

	registry.mu.RLock()
	definition, ok := registry.definitions[key]
	registry.mu.RUnlock()
	if !ok {
		return Definition{}, false
	}
	return cloneDefinition(definition), true
}

// Keys returns registered keys in deterministic lexical order.
func (registry *Registry) Keys() []ResourceKey {
	if registry == nil {
		return nil
	}

	registry.mu.RLock()
	keys := make([]ResourceKey, 0, len(registry.definitions))
	for key := range registry.definitions {
		keys = append(keys, key)
	}
	registry.mu.RUnlock()

	sort.Slice(keys, func(left, right int) bool { return keys[left] < keys[right] })
	return keys
}

// Seal permanently disables registration. It is safe to call more than once.
func (registry *Registry) Seal() {
	if registry == nil {
		return
	}
	registry.mu.Lock()
	registry.sealed = true
	registry.mu.Unlock()
}

// Sealed reports whether the registry accepts new definitions.
func (registry *Registry) Sealed() bool {
	if registry == nil {
		return true
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	return registry.sealed
}

// ValidateDefinition checks every startup invariant independent of a concrete
// persistence implementation.
func ValidateDefinition(ctx context.Context, definition Definition) error {
	if definition.Contract != ContractDefinitionV1 {
		return invalidDefinition("contract", "must be clear.crud.definition.v1")
	}
	if !validKey(string(definition.Key)) {
		return invalidDefinition("key", "must match ^[a-z][a-z0-9_]{0,63}$")
	}
	if definition.Labels.Title == "" {
		return invalidDefinition("labels.title", "is required")
	}
	if definition.Labels.Singular == "" {
		return invalidDefinition("labels.singular", "is required")
	}
	if definition.Permissions.Read == "" {
		return invalidDefinition("permissions.read", "is required")
	}
	if isNil(definition.Source) {
		return invalidDefinition("source", "is required")
	}
	if err := validateScope(definition.Scope); err != nil {
		return err
	}
	fieldIndex, err := validateFields(definition.Fields)
	if err != nil {
		return err
	}
	if err := validateList(definition.List, fieldIndex); err != nil {
		return err
	}
	if err := validatePresentation(definition.Presentation); err != nil {
		return err
	}
	if err := validateDeletePolicy(definition); err != nil {
		return err
	}

	capabilities := definition.Source.Capabilities(ctx)
	if err := validateCapabilities(definition, capabilities); err != nil {
		return err
	}
	return nil
}

func validateScope(scope ScopeRequirements) error {
	seen := make(map[string]struct{}, len(scope.Keys))
	for index, key := range scope.Keys {
		if !validKey(key) {
			return invalidDefinition(fmt.Sprintf("scope.keys[%d]", index), "must be a valid key")
		}
		if _, duplicate := seen[key]; duplicate {
			return invalidDefinition("scope.keys", "contains a duplicate key")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateFields(fields []Field) (map[FieldKey]Field, error) {
	if len(fields) == 0 {
		return nil, invalidDefinition("fields", "must contain at least one field")
	}

	index := make(map[FieldKey]Field, len(fields))
	visible := false
	for position, field := range fields {
		path := fmt.Sprintf("fields[%d]", position)
		if !validKey(string(field.Key)) {
			return nil, invalidDefinition(path+".key", "must be a valid key")
		}
		if _, duplicate := index[field.Key]; duplicate {
			return nil, invalidDefinition("fields", "contains a duplicate key")
		}
		if field.Label == "" {
			return nil, invalidDefinition(path+".label", "is required")
		}
		if !knownFieldType(field.Type) {
			return nil, invalidDefinition(path+".type", "is unknown")
		}
		if field.MinLength < 0 || field.MaxLength < 0 || (field.MaxLength > 0 && field.MinLength > field.MaxLength) {
			return nil, invalidDefinition(path+".length", "has an invalid range")
		}
		if field.Visible {
			visible = true
		}
		index[field.Key] = field
	}
	if !visible {
		return nil, invalidDefinition("fields", "must contain at least one visible field")
	}
	for position, field := range fields {
		path := fmt.Sprintf("fields[%d]", position)
		if field.Type == FieldEnum {
			if len(field.Enum) == 0 {
				return nil, invalidDefinition(path+".enum", "is required for enum fields")
			}
			if err := validateOptions(path+".enum", field.Enum); err != nil {
				return nil, err
			}
		} else if len(field.Enum) != 0 {
			return nil, invalidDefinition(path+".enum", "is only valid for enum fields")
		}
		if field.Type == FieldLookup {
			if err := validateLookup(path, field.Lookup, index); err != nil {
				return nil, err
			}
		} else if field.Lookup != nil {
			return nil, invalidDefinition(path+".lookup", "is only valid for lookup fields")
		}
	}
	return index, nil
}

func validateOptions(path string, options []Option) error {
	seen := make(map[string]struct{}, len(options))
	for index, option := range options {
		if option.Label == "" {
			return invalidDefinition(fmt.Sprintf("%s[%d].label", path, index), "is required")
		}
		if !allowedValue(option.Value) {
			return invalidDefinition(fmt.Sprintf("%s[%d].value", path, index), "must be nil, string, bool, or int64")
		}
		key := fmt.Sprintf("%T:%v", option.Value, option.Value)
		if _, duplicate := seen[key]; duplicate {
			return invalidDefinition(path, "contains a duplicate value")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateLookup(path string, lookup *LookupDefinition, fields map[FieldKey]Field) error {
	if lookup == nil {
		return invalidDefinition(path+".lookup", "is required for lookup fields")
	}
	if !validKey(string(lookup.Resource)) {
		return invalidDefinition(path+".lookup.resource", "must be a valid resource key")
	}
	if lookup.ValueField == "" || lookup.LabelField == "" {
		return invalidDefinition(path+".lookup", "requires value and label fields")
	}
	if lookup.PageSize == 0 || lookup.PageSize > 100 {
		return invalidDefinition(path+".lookup.pageSize", "must be between 1 and 100")
	}
	seen := make(map[FieldKey]struct{}, len(lookup.Dependencies))
	for index, dependency := range lookup.Dependencies {
		if _, exists := fields[dependency]; !exists {
			return invalidDefinition(fmt.Sprintf("%s.lookup.dependencies[%d]", path, index), "references an unknown field")
		}
		if _, duplicate := seen[dependency]; duplicate {
			return invalidDefinition(path+".lookup.dependencies", "contains a duplicate field")
		}
		seen[dependency] = struct{}{}
	}
	return nil
}

func validateList(list ListDefinition, fields map[FieldKey]Field) error {
	if len(list.Columns) == 0 {
		return invalidDefinition("list.columns", "must contain at least one field")
	}
	if err := validateFieldReferences("list.columns", list.Columns, fields); err != nil {
		return err
	}
	if err := validateFieldReferences("list.searchable", list.Searchable, fields); err != nil {
		return err
	}
	if err := validateFieldReferences("list.sortable", list.Sortable, fields); err != nil {
		return err
	}
	if len(list.DefaultSort) == 0 {
		return invalidDefinition("list.defaultSort", "must contain a stable default ordering")
	}
	sortable := make(map[FieldKey]struct{}, len(list.Sortable))
	for _, key := range list.Sortable {
		sortable[key] = struct{}{}
	}
	for index, term := range list.DefaultSort {
		if _, exists := sortable[term.Field]; !exists {
			return invalidDefinition(fmt.Sprintf("list.defaultSort[%d]", index), "must reference a sortable field")
		}
		if term.Direction != SortAscending && term.Direction != SortDescending {
			return invalidDefinition(fmt.Sprintf("list.defaultSort[%d].direction", index), "must be asc or desc")
		}
	}
	if list.Pagination.Mode != PageModeOffset && list.Pagination.Mode != PageModeCursor {
		return invalidDefinition("list.pagination.mode", "must be offset or cursor")
	}
	if list.Pagination.DefaultSize != 25 {
		return invalidDefinition("list.pagination.defaultSize", "must be 25")
	}
	if len(list.Pagination.AllowedSizes) != 3 || list.Pagination.AllowedSizes[0] != 25 || list.Pagination.AllowedSizes[1] != 50 || list.Pagination.AllowedSizes[2] != 100 {
		return invalidDefinition("list.pagination.allowedSizes", "must be [25, 50, 100]")
	}
	return nil
}

func validateFieldReferences(path string, references []FieldKey, fields map[FieldKey]Field) error {
	seen := make(map[FieldKey]struct{}, len(references))
	for index, key := range references {
		if _, exists := fields[key]; !exists {
			return invalidDefinition(fmt.Sprintf("%s[%d]", path, index), "references an unknown field")
		}
		if _, duplicate := seen[key]; duplicate {
			return invalidDefinition(path, "contains a duplicate field")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validatePresentation(presentation Presentation) error {
	if presentation.Collection != CollectionAuto && presentation.Collection != CollectionTable && presentation.Collection != CollectionCards && presentation.Collection != CollectionList {
		return invalidDefinition("presentation.collection", "must be auto, table, cards, or list")
	}
	if presentation.Density != DensityCompact && presentation.Density != DensityComfortable {
		return invalidDefinition("presentation.density", "must be compact or comfortable")
	}
	return nil
}

func validateDeletePolicy(definition Definition) error {
	mode := definition.Delete.Mode
	if mode != DeleteModeNone && mode != DeleteModeArchive && mode != DeleteModeHardDelete {
		return invalidDefinition("delete.mode", "must be none, archive, or hard_delete")
	}
	if mode == DeleteModeNone && definition.Permissions.Delete != "" {
		return invalidDefinition("permissions.delete", "must be empty when delete mode is none")
	}
	if mode != DeleteModeNone && definition.Permissions.Delete == "" {
		return invalidDefinition("permissions.delete", "is required when delete mode is enabled")
	}
	return nil
}

func validateCapabilities(definition Definition, capabilities Capabilities) error {
	if definition.List.Pagination.Mode == PageModeOffset && !capabilities.Has(CapabilityOffsetPage) {
		return invalidDefinition("source.capabilities", "must include offset_page")
	}
	if definition.List.Pagination.Mode == PageModeCursor && !capabilities.Has(CapabilityCursorPage) {
		return invalidDefinition("source.capabilities", "must include cursor_page")
	}
	if definition.List.Pagination.Total && !capabilities.Has(CapabilityTotalCount) {
		return invalidDefinition("source.capabilities", "must include total_count")
	}
	lookupRequired := false
	for _, field := range definition.Fields {
		lookupRequired = lookupRequired || field.Type == FieldLookup
	}
	if lookupRequired && !capabilities.Has(CapabilityLookup) {
		return invalidDefinition("source.capabilities", "must include lookup")
	}

	mutable := definition.Permissions.Create != "" || definition.Permissions.Update != "" || definition.Delete.Mode != DeleteModeNone
	if !mutable {
		return nil
	}
	if definition.Concurrency.Mode != ConcurrencyVersion {
		return invalidDefinition("concurrency.mode", "must be version for mutable resources")
	}
	if isNil(definition.UOW) {
		return invalidDefinition("uow", "is required for mutable resources")
	}
	if !capabilities.Has(CapabilityAtomicVersion) || !capabilities.Has(CapabilityUnitOfWork) {
		return invalidDefinition("source.capabilities", "must include atomic_version and unit_of_work for mutable resources")
	}
	if definition.Delete.Mode == DeleteModeArchive && !capabilities.Has(CapabilityArchive) {
		return invalidDefinition("source.capabilities", "must include archive")
	}
	if definition.Delete.Mode == DeleteModeHardDelete && !capabilities.Has(CapabilityHardDelete) {
		return invalidDefinition("source.capabilities", "must include hard_delete")
	}
	return nil
}

func knownFieldType(fieldType FieldType) bool {
	switch fieldType {
	case FieldString, FieldText, FieldInteger, FieldDecimal, FieldBoolean, FieldDate, FieldDateTime, FieldEmail, FieldPhone, FieldEnum, FieldLookup:
		return true
	default:
		return false
	}
}

func allowedValue(value Value) bool {
	switch value.(type) {
	case nil, string, bool, int64:
		return true
	default:
		return false
	}
}

func validKey(key string) bool {
	return keyPattern.MatchString(key)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func invalidDefinition(path, reason string) error {
	return &DefinitionError{Path: path, Reason: reason}
}

func cloneDefinition(definition Definition) Definition {
	clone := definition
	clone.Scope.Keys = append([]string(nil), definition.Scope.Keys...)
	clone.Fields = make([]Field, len(definition.Fields))
	for index, field := range definition.Fields {
		clone.Fields[index] = field
		clone.Fields[index].Enum = append([]Option(nil), field.Enum...)
		if field.Lookup != nil {
			lookup := *field.Lookup
			lookup.Dependencies = append([]FieldKey(nil), field.Lookup.Dependencies...)
			clone.Fields[index].Lookup = &lookup
		}
	}
	clone.List.Columns = append([]FieldKey(nil), definition.List.Columns...)
	clone.List.Searchable = append([]FieldKey(nil), definition.List.Searchable...)
	clone.List.Sortable = append([]FieldKey(nil), definition.List.Sortable...)
	clone.List.DefaultSort = append([]Sort(nil), definition.List.DefaultSort...)
	clone.List.Pagination.AllowedSizes = append([]uint16(nil), definition.List.Pagination.AllowedSizes...)
	return clone
}
