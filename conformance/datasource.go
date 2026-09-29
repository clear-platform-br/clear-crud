// Package conformance provides reusable behavioral tests for clear.crud
// DataSource adapters. Adapters own their test database and pass a fixture;
// this package never selects a driver, a DSN, or a physical schema.
package conformance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	crud "github.com/clear-platform-br/clear-crud"
)

// Factory creates an isolated adapter fixture for one conformance subtest.
// It must return an empty store. A real adapter should create its actual
// SQLite, PostgreSQL, MySQL, or MariaDB test schema here, never a mock.
type Factory func(testing.TB) Fixture

// Fixture supplies the adapter-specific facts the generic suite cannot infer:
// two trusted scopes, valid mutations, and a normalized base list query.
// NewMutation must return a distinct valid mutation for each label.
type Fixture struct {
	Source      crud.DataSource
	UnitOfWork  crud.UnitOfWork
	ScopeA      crud.Scope
	ScopeB      crud.Scope
	ListQuery   crud.Query
	LookupQuery *crud.LookupQuery
	NewMutation func(label string) crud.Mutation
	Close       func() error
}

// TestDataSource runs the clear.crud v1 behavioral contract for a mutable
// DataSource. It tests only capabilities declared by the adapter. Call it from
// the adapter repository's normal Go test suite:
//
//	func TestPostgresDataSource(t *testing.T) {
//		conformance.TestDataSource(t, newPostgresFixture)
//	}
func TestDataSource(t *testing.T, factory Factory) {
	t.Helper()
	if factory == nil {
		t.Fatal("clear.crud conformance: nil fixture factory")
	}

	t.Run("canceled context is honored", func(t *testing.T) {
		fixture := newFixture(t, factory)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := fixture.Source.List(ctx, fixture.ScopeA, fixture.ListQuery); err == nil {
			t.Fatal("List accepted a canceled context")
		}
	})

	t.Run("scope prevents cross-boundary reads", func(t *testing.T) {
		fixture := newFixture(t, factory)
		record := create(t, fixture, fixture.ScopeA, "scope-a")
		got, err := fixture.Source.Get(context.Background(), fixture.ScopeA, record.ID)
		if err != nil {
			t.Fatalf("Get in owning scope: %v", err)
		}
		if got.ID != record.ID {
			t.Fatalf("Get returned ID %q, want %q", got.ID, record.ID)
		}
		_, err = fixture.Source.Get(context.Background(), fixture.ScopeB, record.ID)
		requireCode(t, err, crud.ErrorNotFound)
		_, err = fixture.Source.Get(context.Background(), fixture.ScopeA, crud.RecordID("missing"))
		requireCode(t, err, crud.ErrorNotFound)

		page, err := fixture.Source.List(context.Background(), fixture.ScopeB, fixture.ListQuery)
		if err != nil {
			t.Fatalf("List in another scope: %v", err)
		}
		for _, listed := range page.Records {
			if listed.ID == record.ID {
				t.Fatal("List leaked a record from another scope")
			}
		}
	})

	t.Run("pagination follows declared capabilities", func(t *testing.T) {
		fixture := newFixture(t, factory)
		capabilities := fixture.Source.Capabilities(context.Background())
		if !capabilities.Has(crud.CapabilityOffsetPage) && !capabilities.Has(crud.CapabilityCursorPage) {
			t.Skip("adapter does not declare pagination")
		}
		first := create(t, fixture, fixture.ScopeA, "page-first")
		second := create(t, fixture, fixture.ScopeA, "page-second")
		if capabilities.Has(crud.CapabilityOffsetPage) {
			query := fixture.ListQuery
			query.Page = crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 1}
			page, err := fixture.Source.List(context.Background(), fixture.ScopeA, query)
			if err != nil {
				t.Fatalf("offset List: %v", err)
			}
			if len(page.Records) != 1 || page.Page != 1 || page.Size != 1 {
				t.Fatalf("invalid offset page: %#v", page)
			}
			if capabilities.Has(crud.CapabilityTotalCount) && (page.Total == nil || *page.Total != 2) {
				t.Fatalf("invalid total count: %#v", page.Total)
			}
			if page.Records[0].ID != first.ID && page.Records[0].ID != second.ID {
				t.Fatal("offset page returned an unknown record")
			}
		}
		if capabilities.Has(crud.CapabilityCursorPage) {
			query := fixture.ListQuery
			query.Page = crud.PageRequest{Mode: crud.PageModeCursor, Size: 1}
			page, err := fixture.Source.List(context.Background(), fixture.ScopeA, query)
			if err != nil {
				t.Fatalf("cursor List: %v", err)
			}
			if len(page.Records) != 1 || page.Size != 1 || page.NextCursor == "" {
				t.Fatalf("invalid cursor page: %#v", page)
			}
		}
	})

	t.Run("optimistic versions are atomic", func(t *testing.T) {
		fixture := newFixture(t, factory)
		if !fixture.Source.Capabilities(context.Background()).Has(crud.CapabilityAtomicVersion) {
			t.Skip("adapter does not declare atomic versions")
		}
		record := create(t, fixture, fixture.ScopeA, "versioned")
		updated, err := fixture.Source.Update(context.Background(), fixture.ScopeA, record.ID, record.Version, fixture.NewMutation("updated"))
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Version == record.Version {
			t.Fatalf("Update preserved stale version %d", record.Version)
		}
		_, err = fixture.Source.Update(context.Background(), fixture.ScopeA, record.ID, record.Version, fixture.NewMutation("stale"))
		requireCode(t, err, crud.ErrorConflict)

		raced := create(t, fixture, fixture.ScopeA, "raced")
		results := make(chan error, 2)
		var group sync.WaitGroup
		for index := range 2 {
			group.Add(1)
			go func(index int) {
				defer group.Done()
				_, err := fixture.Source.Update(context.Background(), fixture.ScopeA, raced.ID, raced.Version, fixture.NewMutation(fmt.Sprintf("race-%d", index)))
				results <- err
			}(index)
		}
		group.Wait()
		close(results)
		successes := 0
		conflicts := 0
		for err := range results {
			if err == nil {
				successes++
				continue
			}
			if codeOf(err) == crud.ErrorConflict {
				conflicts++
				continue
			}
			t.Fatalf("concurrent Update returned %v", err)
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("concurrent Update results: %d successes, %d conflicts", successes, conflicts)
		}
	})

	t.Run("delete policy matches capabilities", func(t *testing.T) {
		fixture := newFixture(t, factory)
		capabilities := fixture.Source.Capabilities(context.Background())
		record := create(t, fixture, fixture.ScopeA, "delete-denied")
		err := fixture.Source.Delete(context.Background(), fixture.ScopeA, record.ID, record.Version, crud.DeleteModeNone)
		requireCode(t, err, crud.ErrorDeleteRestricted)
		for _, policy := range []struct {
			capability crud.Capability
			mode       crud.DeleteMode
			label      string
		}{
			{crud.CapabilityArchive, crud.DeleteModeArchive, "archive"},
			{crud.CapabilityHardDelete, crud.DeleteModeHardDelete, "hard delete"},
		} {
			if !capabilities.Has(policy.capability) {
				continue
			}
			t.Run(policy.label, func(t *testing.T) {
				record := create(t, fixture, fixture.ScopeA, "delete-"+policy.label)
				if err := fixture.Source.Delete(context.Background(), fixture.ScopeA, record.ID, record.Version, policy.mode); err != nil {
					t.Fatalf("Delete %s: %v", policy.label, err)
				}
				_, err := fixture.Source.Get(context.Background(), fixture.ScopeA, record.ID)
				requireCode(t, err, crud.ErrorNotFound)
				if policy.mode == crud.DeleteModeArchive {
					page, err := fixture.Source.List(context.Background(), fixture.ScopeA, fixture.ListQuery)
					if err != nil {
						t.Fatalf("List after archive: %v", err)
					}
					for _, listed := range page.Records {
						if listed.ID == record.ID {
							t.Fatal("archived record appeared in normal List")
						}
					}
					if _, err := fixture.Source.Update(context.Background(), fixture.ScopeA, record.ID, record.Version, fixture.NewMutation("archived-update")); err == nil {
						t.Fatal("archived record accepted a normal Update")
					}
				}
			})
		}
	})

	t.Run("lookup is bounded when declared", func(t *testing.T) {
		fixture := newFixture(t, factory)
		if !fixture.Source.Capabilities(context.Background()).Has(crud.CapabilityLookup) {
			t.Skip("adapter does not declare lookup")
		}
		if fixture.LookupQuery == nil {
			t.Fatal("lookup capability requires Fixture.LookupQuery")
		}
		// The conformance fixture is intentionally empty. Seed one bounded
		// option through the adapter's own mutation path before reading it.
		create(t, fixture, fixture.ScopeA, "lookup-option")
		page, err := fixture.Source.Lookup(context.Background(), fixture.ScopeA, *fixture.LookupQuery)
		if err != nil {
			t.Fatalf("Lookup: %v", err)
		}
		if len(page.Options) == 0 {
			t.Fatal("Lookup returned no options")
		}
		for _, option := range page.Options {
			if option.Label == "" || option.Value == nil {
				t.Fatalf("Lookup returned incomplete option: %#v", option)
			}
		}
	})

	t.Run("transaction rolls back mutation when declared", func(t *testing.T) {
		fixture := newFixture(t, factory)
		if !fixture.Source.Capabilities(context.Background()).Has(crud.CapabilityUnitOfWork) {
			t.Skip("adapter does not declare unit of work")
		}
		if fixture.UnitOfWork == nil {
			t.Fatal("unit_of_work capability requires Fixture.UnitOfWork")
		}
		var created crud.Record
		rollback := errors.New("conformance rollback")
		err := fixture.UnitOfWork.Within(context.Background(), func(ctx context.Context) error {
			var err error
			created, err = fixture.Source.Create(ctx, fixture.ScopeA, fixture.NewMutation("rollback"))
			if err != nil {
				return err
			}
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatalf("Within error = %v, want rollback error", err)
		}
		_, err = fixture.Source.Get(context.Background(), fixture.ScopeA, created.ID)
		requireCode(t, err, crud.ErrorNotFound)
	})
}

func newFixture(t *testing.T, factory Factory) Fixture {
	t.Helper()
	fixture := factory(t)
	if fixture.Source == nil {
		t.Fatal("clear.crud conformance: fixture Source is nil")
	}
	if len(fixture.ScopeA) == 0 || len(fixture.ScopeB) == 0 || sameScope(fixture.ScopeA, fixture.ScopeB) {
		t.Fatal("clear.crud conformance: fixture requires two distinct non-empty trusted scopes")
	}
	if fixture.NewMutation == nil {
		t.Fatal("clear.crud conformance: fixture NewMutation is nil")
	}
	if fixture.Close != nil {
		t.Cleanup(func() {
			if err := fixture.Close(); err != nil {
				t.Errorf("clear.crud conformance fixture close: %v", err)
			}
		})
	}
	return fixture
}

func create(t *testing.T, fixture Fixture, scope crud.Scope, label string) crud.Record {
	t.Helper()
	record, err := fixture.Source.Create(context.Background(), scope, fixture.NewMutation(label))
	if err != nil {
		t.Fatalf("Create %q: %v", label, err)
	}
	if record.ID == "" || record.Version == 0 {
		t.Fatalf("Create %q returned incomplete record: %#v", label, record)
	}
	return record
}

func requireCode(t *testing.T, err error, want crud.ErrorCode) {
	t.Helper()
	if got := codeOf(err); got != want {
		t.Fatalf("error code = %q for %v, want %q", got, err, want)
	}
}

func codeOf(err error) crud.ErrorCode {
	var public *crud.Error
	if errors.As(err, &public) && public != nil {
		return public.Code
	}
	return ""
}

func sameScope(left, right crud.Scope) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
