package crud

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
)

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// DefinitionError identifies an invalid declaration during application startup.
// It never crosses an operator-facing transport boundary.
type DefinitionError struct {
	Path   string
	Reason string
}

func (err *DefinitionError) Error() string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("invalid CRUD definition at %s: %s", err.Path, err.Reason)
}

// ValidateDefinition checks that a resource is structurally valid and that its
// declared persistence guarantees can satisfy its public behavior.
func ValidateDefinition(ctx context.Context, definition Definition) error {
	if definition.Contract != ContractDefinitionV1 {
		return invalidDefinition("contract", "must be clear.crud.definition.v1")
	}
	if !validKey(string(definition.Key)) {
		return invalidDefinition("key", "must be a lowercase identifier")
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
	fields, err := validateFields(definition.Fields)
	if err != nil {
		return err
	}
	if err := validateList(definition.List, fields); err != nil {
		return err
	}
	if err := validatePresentation(definition.Presentation); err != nil {
		return err
	}
	if err := validateDeletePolicy(definition); err != nil {
		return err
	}
	if err := validateDetails(definition.Details); err != nil {
		return err
	}
	return validateCapabilities(definition, definition.Source.Capabilities(ctx))
}

func validateDetails(details []DetailDefinition) error {
	seen := make(map[DetailKey]struct{}, len(details))
	for index, detail := range details {
		path := fmt.Sprintf("details[%d]", index)
		if !validKey(string(detail.Key)) {
			return invalidDefinition(path+".key", "must be a lowercase identifier")
		}
		if _, exists := seen[detail.Key]; exists {
			return invalidDefinition(path+".key", "must not be duplicated")
		}
		seen[detail.Key] = struct{}{}
		if !validKey(string(detail.Resource)) {
			return invalidDefinition(path+".resource", "must be a lowercase identifier")
		}
		if !validKey(string(detail.ParentField)) {
			return invalidDefinition(path+".parent_field", "must be a lowercase identifier")
		}
		if detail.Maximum == 0 || detail.Maximum > maxDetailRecords {
			return invalidDefinition(path+".maximum", "must be between 1 and 100")
		}
		if detail.Minimum > detail.Maximum {
			return invalidDefinition(path+".minimum", "must not exceed maximum")
		}
	}
	return nil
}

func validateScope(scope ScopeRequirements) error {
	seen := make(map[string]struct{}, len(scope.Keys))
	for index, key := range scope.Keys {
		if !validKey(key) {
			return invalidDefinition(fmt.Sprintf("scope.keys[%d]", index), "must be a lowercase identifier")
		}
		if _, exists := seen[key]; exists {
			return invalidDefinition("scope.keys", "must not contain duplicates")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validatePresentation(presentation Presentation) error {
	if presentation.Collection != CollectionAuto && presentation.Collection != CollectionTable &&
		presentation.Collection != CollectionCards && presentation.Collection != CollectionList {
		return invalidDefinition("presentation.collection", "is unknown")
	}
	if presentation.Density != DensityCompact && presentation.Density != DensityComfortable {
		return invalidDefinition("presentation.density", "is unknown")
	}
	return nil
}

func validateDeletePolicy(definition Definition) error {
	if definition.Delete.Mode != DeleteModeNone && definition.Delete.Mode != DeleteModeArchive && definition.Delete.Mode != DeleteModeHardDelete {
		return invalidDefinition("delete.mode", "is unknown")
	}
	if definition.Permissions.Delete != "" && definition.Delete.Mode == DeleteModeNone {
		return invalidDefinition("permissions.delete", "requires a delete mode")
	}
	if definition.Permissions.Delete == "" && definition.Delete.Mode != DeleteModeNone {
		return invalidDefinition("permissions.delete", "is required when delete mode is enabled")
	}
	return nil
}

func validateCapabilities(definition Definition, capabilities Capabilities) error {
	if definition.List.Pagination.Mode == PageModeOffset && !capabilities.Has(CapabilityOffsetPage) {
		return invalidDefinition("source.capabilities", "must support offset pagination")
	}
	if definition.List.Pagination.Mode == PageModeCursor && !capabilities.Has(CapabilityCursorPage) {
		return invalidDefinition("source.capabilities", "must support cursor pagination")
	}
	if definition.List.Pagination.Total && !capabilities.Has(CapabilityTotalCount) {
		return invalidDefinition("source.capabilities", "must support total count")
	}
	for _, field := range definition.Fields {
		if field.Type == FieldLookup && !capabilities.Has(CapabilityLookup) {
			return invalidDefinition("source.capabilities", "must support lookups")
		}
	}

	mutable := definition.Permissions.Create != "" || definition.Permissions.Update != "" || definition.Delete.Mode != DeleteModeNone
	if !mutable {
		return nil
	}
	if definition.Concurrency.Mode != ConcurrencyVersion {
		return invalidDefinition("concurrency.mode", "must require a version for mutable resources")
	}
	if isNil(definition.UOW) {
		return invalidDefinition("uow", "is required for mutable resources")
	}
	if !capabilities.Has(CapabilityAtomicVersion) || !capabilities.Has(CapabilityUnitOfWork) {
		return invalidDefinition("source.capabilities", "must support atomic versioning and unit of work")
	}
	if definition.Delete.Mode == DeleteModeArchive && !capabilities.Has(CapabilityArchive) {
		return invalidDefinition("source.capabilities", "must support archive deletion")
	}
	if definition.Delete.Mode == DeleteModeHardDelete && !capabilities.Has(CapabilityHardDelete) {
		return invalidDefinition("source.capabilities", "must support hard deletion")
	}
	return nil
}

func validKey(key string) bool { return keyPattern.MatchString(key) }

func isNil(value any) bool {
	if value == nil {
		return true
	}
	kind := reflect.ValueOf(value).Kind()
	return (kind == reflect.Chan || kind == reflect.Func || kind == reflect.Interface || kind == reflect.Map || kind == reflect.Pointer || kind == reflect.Slice) && reflect.ValueOf(value).IsNil()
}

func invalidDefinition(path, reason string) error {
	return &DefinitionError{Path: path, Reason: reason}
}
