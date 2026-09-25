package crud

import (
	"context"
	"errors"
)

// Dependencies contains the host ports required by Service. The host owns
// identity, scope, authorization, auditing, translation, and time.
type Dependencies struct {
	Registry   *Registry
	Principal  PrincipalProvider
	Scope      ScopeProvider
	Authorizer Authorizer
	Audit      AuditSink
	Translator Translator
	Clock      Clock
}

// Service applies the public CRUD contract before delegating to a DataSource.
type Service struct {
	registry   *Registry
	principal  PrincipalProvider
	scope      ScopeProvider
	authorizer Authorizer
	audit      AuditSink
	translator Translator
	clock      Clock
}

// PublicDefinition contains only renderer-safe resource metadata. It never
// exposes permission keys, scope, persistence, hooks, or lifecycle internals.
type PublicDefinition struct {
	Key          ResourceKey
	Labels       Labels
	Fields       []Field
	Details      []PublicDetailDefinition
	List         ListDefinition
	Presentation Presentation
	Actions      []Action
}

// NewService validates the host ports once during startup.
func NewService(dependencies Dependencies) (*Service, error) {
	switch {
	case dependencies.Registry == nil:
		return nil, errors.New("crud service requires a registry")
	case isNil(dependencies.Principal):
		return nil, errors.New("crud service requires a principal provider")
	case isNil(dependencies.Scope):
		return nil, errors.New("crud service requires a scope provider")
	case isNil(dependencies.Authorizer):
		return nil, errors.New("crud service requires an authorizer")
	case isNil(dependencies.Audit):
		return nil, errors.New("crud service requires an audit sink")
	case isNil(dependencies.Translator):
		return nil, errors.New("crud service requires a translator")
	case isNil(dependencies.Clock):
		return nil, errors.New("crud service requires a clock")
	}
	service := &Service{
		registry: dependencies.Registry, principal: dependencies.Principal,
		scope: dependencies.Scope, authorizer: dependencies.Authorizer,
		audit: dependencies.Audit, translator: dependencies.Translator,
		clock: dependencies.Clock,
	}
	if err := validateMasterDetailDefinitions(dependencies.Registry); err != nil {
		return nil, err
	}
	return service, nil
}

// Definition returns a renderer-safe definition for a principal that can read
// the resource.
func (service *Service) Definition(ctx context.Context, key ResourceKey) (PublicDefinition, error) {
	state, err := service.resolveRead(ctx, key)
	if err != nil {
		return PublicDefinition{}, err
	}
	return publicDefinition(ctx, state, service.authorizer, service.registry), nil
}

type readState struct {
	definition Definition
	principal  Principal
	scope      Scope
}

func (service *Service) resolveRead(ctx context.Context, key ResourceKey) (readState, error) {
	if service == nil {
		return readState{}, unavailable(nil)
	}
	definition, ok := service.registry.Get(key)
	if !ok {
		return readState{}, publicError(ErrorNotFound, "crud.error.not_found", nil)
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
	if err := service.authorizer.Authorize(ctx, principal, key, ActionRead, nil); err != nil {
		return readState{}, publicError(ErrorForbidden, "crud.error.forbidden", err)
	}
	return readState{definition: definition, principal: principal, scope: cloneScope(scope)}, nil
}

const (
	maxQuerySearchRunes = 256
	maxQueryFilters     = 20
	maxQuerySortTerms   = 8
	maxCursorBytes      = 512
)

func validateTrustedScope(requirements ScopeRequirements, scope Scope) error {
	for _, key := range requirements.Keys {
		if scope[key] == "" {
			return publicError(ErrorForbidden, "crud.error.forbidden", nil)
		}
	}
	return nil
}

// PublicDetailDefinition contains renderer-safe metadata for one child collection.
type PublicDetailDefinition struct {
	Key         DetailKey
	Resource    ResourceKey
	Fields      []Field
	Minimum     uint16
	Maximum     uint16
	AllowCreate bool
	AllowUpdate bool
	AllowDelete bool
}

func publicDefinition(ctx context.Context, state readState, authorizer Authorizer, registry *Registry) PublicDefinition {
	definition := PublicDefinition{
		Key:          state.definition.Key,
		Labels:       state.definition.Labels,
		List:         cloneDefinition(state.definition).List,
		Presentation: state.definition.Presentation,
	}
	for _, field := range state.definition.Fields {
		if field.Visible {
			definition.Fields = append(definition.Fields, field)
		}
	}
	for _, action := range []Action{ActionCreate, ActionUpdate, ActionDelete, ActionHelp} {
		if actionEnabled(state.definition.Permissions, action) &&
			authorizer.Authorize(ctx, state.principal, state.definition.Key, action, nil) == nil {
			definition.Actions = append(definition.Actions, action)
		}
	}
	for _, detail := range state.definition.Details {
		child, ok := registry.Get(detail.Resource)
		if !ok || authorizer.Authorize(ctx, state.principal, child.Key, ActionRead, nil) != nil {
			continue
		}
		public := PublicDetailDefinition{Key: detail.Key, Resource: detail.Resource, Minimum: detail.Minimum, Maximum: detail.Maximum}
		for _, field := range child.Fields {
			if field.Key != detail.ParentField && field.Visible {
				public.Fields = append(public.Fields, field)
			}
		}
		public.AllowCreate = detail.AllowCreate && actionEnabled(child.Permissions, ActionCreate) && authorizer.Authorize(ctx, state.principal, child.Key, ActionCreate, nil) == nil
		public.AllowUpdate = detail.AllowUpdate && actionEnabled(child.Permissions, ActionUpdate) && authorizer.Authorize(ctx, state.principal, child.Key, ActionUpdate, nil) == nil
		public.AllowDelete = detail.AllowDelete && actionEnabled(child.Permissions, ActionDelete) && authorizer.Authorize(ctx, state.principal, child.Key, ActionDelete, nil) == nil
		definition.Details = append(definition.Details, public)
	}
	return definition
}

func actionEnabled(permissions Permissions, action Action) bool {
	switch action {
	case ActionCreate:
		return permissions.Create != ""
	case ActionRead:
		return permissions.Read != ""
	case ActionUpdate:
		return permissions.Update != ""
	case ActionDelete:
		return permissions.Delete != ""
	case ActionHelp:
		return permissions.Help != ""
	default:
		return false
	}
}

func cloneScope(scope Scope) Scope {
	clone := make(Scope, len(scope))
	for key, value := range scope {
		clone[key] = value
	}
	return clone
}

func publicError(code ErrorCode, message MessageCode, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func unavailable(cause error) *Error {
	var public *Error
	if errors.As(cause, &public) {
		return public
	}
	return publicError(ErrorTemporarilyUnavailable, "crud.error.unavailable", cause)
}
