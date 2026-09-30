package crud

import (
	"context"
	"errors"
	"fmt"
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
	enriched, err := applySourceMetadata(ctx, definition)
	if err != nil {
		return err
	}
	definition = enriched
	if err := ValidateDefinition(ctx, definition); err != nil {
		return err
	}
	definition, err = prepareDefinitionPatterns(definition)
	if err != nil {
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

func cloneDefinition(definition Definition) Definition {
	clone := definition
	clone.Scope.Keys = append([]string(nil), definition.Scope.Keys...)
	clone.Fields = make([]Field, len(definition.Fields))
	for index, field := range definition.Fields {
		clone.Fields[index] = field
		clone.Fields[index].Default = field.Default
		clone.Fields[index].Enum = append([]Option(nil), field.Enum...)
		if field.BooleanDisplay != nil {
			booleanDisplay := *field.BooleanDisplay
			clone.Fields[index].BooleanDisplay = &booleanDisplay
		}
		if field.Lookup != nil {
			lookup := *field.Lookup
			lookup.Dependencies = append([]FieldKey(nil), field.Lookup.Dependencies...)
			lookup.FixedFilters = cloneFixedLookupFilters(field.Lookup.FixedFilters)
			clone.Fields[index].Lookup = &lookup
		}
	}
	clone.Details = make([]DetailDefinition, len(definition.Details))
	for index, detail := range definition.Details {
		clone.Details[index] = detail
		clone.Details[index].FieldMetadata = append([]DetailFieldMetadataSource(nil), detail.FieldMetadata...)
	}
	clone.Grid.Columns = append([]FieldKey(nil), definition.Grid.Columns...)
	clone.Grid.Searchable = append([]FieldKey(nil), definition.Grid.Searchable...)
	clone.Grid.Sortable = append([]FieldKey(nil), definition.Grid.Sortable...)
	clone.Grid.DefaultSort = append([]Sort(nil), definition.Grid.DefaultSort...)
	clone.Grid.Pagination.AllowedSizes = append([]uint16(nil), definition.Grid.Pagination.AllowedSizes...)
	clone.Form.Fields = append([]FieldKey(nil), definition.Form.Fields...)
	if len(definition.fieldPatterns) != 0 {
		clone.fieldPatterns = make(map[FieldKey]*regexp.Regexp, len(definition.fieldPatterns))
		for key, pattern := range definition.fieldPatterns {
			clone.fieldPatterns[key] = pattern
		}
	}
	return clone
}
