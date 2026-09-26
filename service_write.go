package crud

import "context"

// Create creates a scoped record and audit event in one UnitOfWork.
func (service *Service) Create(ctx context.Context, key ResourceKey, mutation Mutation) (Record, error) {
	state, err := service.resolveAction(ctx, key, ActionCreate)
	if err != nil {
		return Record{}, err
	}
	normalized, err := normalizeMutationForAction(ctx, state.scope, state.definition, ActionCreate, mutation)
	if err != nil {
		return Record{}, err
	}
	detailChanges, err := service.normalizeDetailChanges(ctx, state, normalized.Details)
	if err != nil {
		return Record{}, err
	}
	normalized.Details = nil
	var created Record
	var detailEvents []detailEvent
	err = state.definition.UOW.Within(ctx, func(transaction context.Context) error {
		if state.definition.Hooks.BeforeCreate != nil {
			if err := state.definition.Hooks.BeforeCreate(transaction, state.scope, normalized); err != nil {
				return unavailable(err)
			}
		}
		record, err := state.definition.Source.Create(transaction, state.scope, normalized)
		if err != nil {
			return unavailable(err)
		}
		if err := service.audit.Append(transaction, service.auditEvent(transaction, state, ActionCreate, record.ID, 0, record.Version)); err != nil {
			return unavailable(err)
		}
		detailEvents, err = service.applyDetailChanges(transaction, state, record.ID, detailChanges)
		if err != nil {
			return err
		}
		details, err := service.loadDetails(transaction, state, record.ID)
		if err != nil {
			return err
		}
		created = sanitizeRecord(state.definition, record)
		created.Details = details
		return nil
	})
	if err != nil {
		return Record{}, unavailable(err)
	}
	service.afterCommit(ctx, state.definition, MutationEvent{Action: ActionCreate, Record: created})
	for _, event := range detailEvents {
		service.afterCommit(ctx, event.definition, event.event)
	}
	return created, nil
}

// Update replaces writable field values when the supplied version still matches.
func (service *Service) Update(ctx context.Context, key ResourceKey, id RecordID, version Version, mutation Mutation) (Record, error) {
	if id == "" || version == 0 {
		return Record{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	state, err := service.resolveAction(ctx, key, ActionUpdate)
	if err != nil {
		return Record{}, err
	}
	normalized, err := normalizeMutationForAction(ctx, state.scope, state.definition, ActionUpdate, mutation)
	if err != nil {
		return Record{}, err
	}
	detailChanges, err := service.normalizeDetailChanges(ctx, state, normalized.Details)
	if err != nil {
		return Record{}, err
	}
	normalized.Details = nil
	var updated Record
	var detailEvents []detailEvent
	err = state.definition.UOW.Within(ctx, func(transaction context.Context) error {
		current, err := state.definition.Source.Get(transaction, state.scope, id)
		if err != nil {
			return unavailable(err)
		}
		if err := service.authorizer.Authorize(transaction, state.principal, key, ActionUpdate, &current); err != nil {
			return publicError(ErrorForbidden, "crud.error.forbidden", err)
		}
		if state.definition.Hooks.BeforeUpdate != nil {
			if err := state.definition.Hooks.BeforeUpdate(transaction, state.scope, current, normalized); err != nil {
				return unavailable(err)
			}
		}
		record, err := state.definition.Source.Update(transaction, state.scope, id, version, normalized)
		if err != nil {
			return unavailable(err)
		}
		if err := service.audit.Append(transaction, service.auditEvent(transaction, state, ActionUpdate, id, version, record.Version)); err != nil {
			return unavailable(err)
		}
		detailEvents, err = service.applyDetailChanges(transaction, state, id, detailChanges)
		if err != nil {
			return err
		}
		details, err := service.loadDetails(transaction, state, id)
		if err != nil {
			return err
		}
		updated = sanitizeRecord(state.definition, record)
		updated.Details = details
		return nil
	})
	if err != nil {
		return Record{}, unavailable(err)
	}
	service.afterCommit(ctx, state.definition, MutationEvent{Action: ActionUpdate, Record: updated})
	for _, event := range detailEvents {
		service.afterCommit(ctx, event.definition, event.event)
	}
	return updated, nil
}

// Delete applies the registered archive or hard-delete policy atomically.
func (service *Service) Delete(ctx context.Context, key ResourceKey, id RecordID, version Version) error {
	if id == "" || version == 0 {
		return publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	state, err := service.resolveAction(ctx, key, ActionDelete)
	if err != nil {
		return err
	}
	var deleted Record
	err = state.definition.UOW.Within(ctx, func(transaction context.Context) error {
		current, err := state.definition.Source.Get(transaction, state.scope, id)
		if err != nil {
			return unavailable(err)
		}
		if err := service.authorizer.Authorize(transaction, state.principal, key, ActionDelete, &current); err != nil {
			return publicError(ErrorForbidden, "crud.error.forbidden", err)
		}
		if state.definition.Hooks.BeforeDelete != nil {
			if err := state.definition.Hooks.BeforeDelete(transaction, state.scope, current); err != nil {
				return unavailable(err)
			}
		}
		if err := state.definition.Source.Delete(transaction, state.scope, id, version, state.definition.Delete.Mode); err != nil {
			return unavailable(err)
		}
		afterVersion := Version(0)
		if state.definition.Delete.Mode == DeleteModeArchive {
			afterVersion = version + 1
		}
		if err := service.audit.Append(transaction, service.auditEvent(transaction, state, ActionDelete, id, version, afterVersion)); err != nil {
			return unavailable(err)
		}
		deleted = sanitizeRecord(state.definition, current)
		return nil
	})
	if err != nil {
		return unavailable(err)
	}
	service.afterCommit(ctx, state.definition, MutationEvent{Action: ActionDelete, Record: deleted})
	return nil
}

func (service *Service) resolveAction(ctx context.Context, key ResourceKey, action Action) (readState, error) {
	if service == nil {
		return readState{}, unavailable(nil)
	}
	definition, ok := service.registry.Get(key)
	if !ok {
		return readState{}, publicError(ErrorNotFound, "crud.error.not_found", nil)
	}
	if !actionEnabled(definition.Permissions, action) {
		return readState{}, publicError(ErrorForbidden, "crud.error.forbidden", nil)
	}
	principal, err := service.principal.Principal(ctx)
	if err != nil {
		return readState{}, publicError(ErrorUnauthenticated, "crud.error.unauthenticated", err)
	}
	scope, err := service.scope.Scope(ctx, key)
	if err != nil {
		return readState{}, publicError(ErrorForbidden, "crud.error.forbidden", err)
	}
	if err := validateTrustedScope(definition.Scope, scope); err != nil {
		return readState{}, err
	}
	if err := service.authorizer.Authorize(ctx, principal, key, action, nil); err != nil {
		return readState{}, publicError(ErrorForbidden, "crud.error.forbidden", err)
	}
	return readState{definition: definition, principal: principal, scope: cloneScope(scope)}, nil
}

func (service *Service) auditEvent(ctx context.Context, state readState, action Action, id RecordID, before, after Version) AuditEvent {
	return AuditEvent{
		Resource: state.definition.Key, Action: action, RecordID: id,
		BeforeVersion: before, AfterVersion: after, Principal: state.principal,
		Scope: cloneScope(state.scope), OccurredAt: service.clock.Now(), CorrelationID: CorrelationID(ctx),
	}
}

func (service *Service) afterCommit(ctx context.Context, definition Definition, event MutationEvent) {
	if definition.Hooks.AfterCommit != nil {
		_ = definition.Hooks.AfterCommit(ctx, event)
	}
}
