package crud

import (
	"context"
	"errors"
	"fmt"
)

const maxDetailRecords uint16 = 100

type detailChange struct {
	definition DetailDefinition
	child      Definition
	action     Action
	id         RecordID
	version    Version
	mutation   Mutation
}

type detailEvent struct {
	definition Definition
	event      MutationEvent
}

// validateMasterDetailDefinitions validates relationships after every resource
// has been registered. A child can be reused only as a single-level detail.
func validateMasterDetailDefinitions(registry *Registry) error {
	for _, parentKey := range registry.Keys() {
		parent, _ := registry.Get(parentKey)
		seenResources := make(map[ResourceKey]struct{}, len(parent.Details))
		for index, detail := range parent.Details {
			path := fmt.Sprintf("details[%d]", index)
			if detail.Resource == parent.Key {
				return invalidDefinition(path+".resource", "must not reference its parent resource")
			}
			if _, exists := seenResources[detail.Resource]; exists {
				return invalidDefinition(path+".resource", "must not be declared more than once")
			}
			seenResources[detail.Resource] = struct{}{}
			child, ok := registry.Get(detail.Resource)
			if !ok {
				return invalidDefinition(path+".resource", "must reference a registered resource")
			}
			if len(child.Details) != 0 {
				return invalidDefinition(path+".resource", "must not declare nested details")
			}
			parentField, ok := findField(child.Fields, detail.ParentField)
			if !ok || parentField.Visible || !parentField.ReadOnly || (parentField.Type != FieldString && parentField.Type != FieldLookup) {
				return invalidDefinition(path+".parent_field", "must reference an invisible read-only string or lookup child field")
			}
			if !scopeSubset(child.Scope, parent.Scope) {
				return invalidDefinition(path+".resource", "requires scope keys unavailable from the parent")
			}
			if child.Grid.Pagination.Mode != PageModeOffset || !child.Grid.Pagination.Total || !child.Source.Capabilities(context.Background()).Has(CapabilityTotalCount) {
				return invalidDefinition(path+".resource", "must support offset paging with total count")
			}
			if detailPageSize(child.Grid.Pagination, detail.Maximum) == 0 {
				return invalidDefinition(path+".maximum", "must fit a child page size")
			}
			if err := validateDetailActions(path, detail, child); err != nil {
				return err
			}
		}
	}
	return nil
}

func scopeSubset(child, parent ScopeRequirements) bool {
	for _, childKey := range child.Keys {
		found := false
		for _, parentKey := range parent.Keys {
			if childKey == parentKey {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func validateDetailActions(path string, detail DetailDefinition, child Definition) error {
	for _, action := range []struct {
		enabled bool
		action  Action
	}{
		{detail.AllowCreate, ActionCreate},
		{detail.AllowUpdate, ActionUpdate},
		{detail.AllowDelete, ActionDelete},
	} {
		if action.enabled && !actionEnabled(child.Permissions, action.action) {
			return invalidDefinition(path, "enables a child action unavailable on the resource")
		}
	}
	return nil
}

func detailPageSize(pagination PaginationDefinition, maximum uint16) uint16 {
	for _, size := range pagination.AllowedSizes {
		if size >= maximum {
			return size
		}
	}
	return 0
}

func (service *Service) normalizeDetailChanges(ctx context.Context, state readState, details DetailMutations) ([]detailChange, error) {
	if len(details) == 0 {
		return nil, nil
	}
	definitions := make(map[DetailKey]DetailDefinition, len(state.definition.Details))
	for _, detail := range state.definition.Details {
		definitions[detail.Key] = detail
	}
	changes := make([]detailChange, 0)
	for key, mutations := range details {
		detail, ok := definitions[key]
		if !ok || len(mutations) > int(detail.Maximum) {
			return nil, invalidMutation(nil)
		}
		child, ok := service.registry.Get(detail.Resource)
		if !ok {
			return nil, unavailable(nil)
		}
		for _, mutation := range mutations {
			change, err := normalizeDetailChange(ctx, state.scope, detail, child, mutation)
			if err != nil {
				return nil, detailError(detail, err)
			}
			changes = append(changes, change)
		}
	}
	return changes, nil
}

func normalizeDetailChange(ctx context.Context, scope Scope, detail DetailDefinition, child Definition, mutation DetailMutation) (detailChange, error) {
	change := detailChange{definition: detail, child: child, id: mutation.ID, version: mutation.Version}
	switch {
	case mutation.Delete:
		if !detail.AllowDelete || mutation.ID == "" || mutation.Version == 0 || len(mutation.Fields) != 0 {
			return detailChange{}, invalidMutation(nil)
		}
		change.action = ActionDelete
	case mutation.ID == "":
		if !detail.AllowCreate || mutation.Version != 0 {
			return detailChange{}, invalidMutation(nil)
		}
		normalized, err := normalizeMutationForAction(ctx, scope, child, ActionCreate, Mutation{Fields: mutation.Fields})
		if err != nil {
			return detailChange{}, err
		}
		change.action, change.mutation = ActionCreate, normalized
	default:
		if !detail.AllowUpdate || mutation.Version == 0 {
			return detailChange{}, invalidMutation(nil)
		}
		normalized, err := normalizeMutationForAction(ctx, scope, child, ActionUpdate, Mutation{Fields: mutation.Fields})
		if err != nil {
			return detailChange{}, err
		}
		change.action, change.mutation = ActionUpdate, normalized
	}
	return change, nil
}

func (service *Service) applyDetailChanges(ctx context.Context, parent readState, parentID RecordID, changes []detailChange) ([]detailEvent, error) {
	events := make([]detailEvent, 0, len(changes))
	for _, change := range changes {
		childState := readState{definition: change.child, principal: parent.principal, scope: cloneScope(parent.scope)}
		if err := service.authorizer.Authorize(ctx, parent.principal, change.child.Key, change.action, nil); err != nil {
			return nil, publicError(ErrorForbidden, "crud.error.forbidden", err)
		}
		switch change.action {
		case ActionCreate:
			mutation := change.mutation
			mutation.Fields = cloneFields(mutation.Fields)
			mutation.Fields[change.definition.ParentField] = string(parentID)
			if change.child.Hooks.BeforeCreate != nil {
				if err := change.child.Hooks.BeforeCreate(ctx, childState.scope, mutation); err != nil {
					return nil, unavailable(err)
				}
			}
			record, err := change.child.Source.Create(ctx, childState.scope, mutation)
			if err != nil {
				return nil, detailError(change.definition, unavailable(err))
			}
			if err := service.audit.Append(ctx, service.auditEvent(ctx, childState, ActionCreate, record.ID, 0, record.Version)); err != nil {
				return nil, unavailable(err)
			}
			events = append(events, detailEvent{definition: change.child, event: MutationEvent{Action: ActionCreate, Record: sanitizeRecord(change.child, record)}})
		case ActionUpdate, ActionDelete:
			current, err := change.child.Source.Get(ctx, childState.scope, change.id)
			if err != nil {
				return nil, unavailable(err)
			}
			if !belongsToParent(current, change.definition.ParentField, parentID) {
				return nil, publicError(ErrorNotFound, "crud.error.not_found", nil)
			}
			if err := service.authorizer.Authorize(ctx, parent.principal, change.child.Key, change.action, &current); err != nil {
				return nil, publicError(ErrorForbidden, "crud.error.forbidden", err)
			}
			if change.action == ActionUpdate {
				mutation := change.mutation
				mutation.Fields = cloneFields(mutation.Fields)
				mutation.Fields[change.definition.ParentField] = string(parentID)
				if change.child.Hooks.BeforeUpdate != nil {
					if err := change.child.Hooks.BeforeUpdate(ctx, childState.scope, current, mutation); err != nil {
						return nil, unavailable(err)
					}
				}
				record, err := change.child.Source.Update(ctx, childState.scope, change.id, change.version, mutation)
				if err != nil {
					return nil, detailError(change.definition, unavailable(err))
				}
				if err := service.audit.Append(ctx, service.auditEvent(ctx, childState, ActionUpdate, change.id, change.version, record.Version)); err != nil {
					return nil, unavailable(err)
				}
				events = append(events, detailEvent{definition: change.child, event: MutationEvent{Action: ActionUpdate, Record: sanitizeRecord(change.child, record)}})
				continue
			}
			if change.child.Hooks.BeforeDelete != nil {
				if err := change.child.Hooks.BeforeDelete(ctx, childState.scope, current); err != nil {
					return nil, unavailable(err)
				}
			}
			if err := change.child.Source.Delete(ctx, childState.scope, change.id, change.version, change.child.Delete.Mode); err != nil {
				return nil, unavailable(err)
			}
			after := Version(0)
			if change.child.Delete.Mode == DeleteModeArchive {
				after = change.version + 1
			}
			if err := service.audit.Append(ctx, service.auditEvent(ctx, childState, ActionDelete, change.id, change.version, after)); err != nil {
				return nil, unavailable(err)
			}
			events = append(events, detailEvent{definition: change.child, event: MutationEvent{Action: ActionDelete, Record: sanitizeRecord(change.child, current)}})
		}
	}
	return events, nil
}

func belongsToParent(record Record, field FieldKey, parentID RecordID) bool {
	value, ok := record.Fields[field]
	return ok && fmt.Sprint(value) == string(parentID)
}

func detailError(detail DetailDefinition, err error) error {
	var public *Error
	if !errors.As(err, &public) || public.Code != ErrorValidationFailed || len(public.Fields) == 0 {
		return err
	}
	fields := make(FieldErrors, len(public.Fields))
	for key, message := range public.Fields {
		fields[FieldKey(string(detail.Key)+"."+string(key))] = message
	}
	return &Error{Code: public.Code, Message: public.Message, Fields: fields, Cause: public.Cause}
}

func (service *Service) loadDetails(ctx context.Context, state readState, parentID RecordID) (DetailRecords, error) {
	if len(state.definition.Details) == 0 {
		return nil, nil
	}
	details := make(DetailRecords, len(state.definition.Details))
	for _, detail := range state.definition.Details {
		child, ok := service.registry.Get(detail.Resource)
		if !ok {
			return nil, unavailable(nil)
		}
		if service.authorizer.Authorize(ctx, state.principal, child.Key, ActionRead, nil) != nil {
			continue
		}
		size := detailPageSize(child.Grid.Pagination, detail.Maximum)
		page, err := child.Source.List(ctx, state.scope, Query{
			Filters: []Filter{{Field: detail.ParentField, Operator: FilterEqual, Value: string(parentID)}},
			Page:    PageRequest{Mode: PageModeOffset, Number: 1, Size: size},
		})
		if err != nil {
			return nil, unavailable(err)
		}
		if page.Total == nil || *page.Total < uint64(detail.Minimum) || *page.Total > uint64(detail.Maximum) || len(page.Records) > int(detail.Maximum) {
			return nil, invalidMutation(nil)
		}
		records := make([]Record, len(page.Records))
		for index, record := range page.Records {
			if !belongsToParent(record, detail.ParentField, parentID) {
				return nil, unavailable(nil)
			}
			records[index] = sanitizeRecord(child, record)
		}
		details[detail.Key] = records
	}
	return details, nil
}
