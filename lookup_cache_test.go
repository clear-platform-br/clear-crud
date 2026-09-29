package crud

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryLookupCacheBoundsClonesAndExpires(t *testing.T) {
	cache := NewMemoryLookupCache(MemoryLookupCacheOptions{MaxEntries: 1, TTL: time.Minute})
	page := LookupPage{Options: []LookupOption{{Value: "1", Label: "One"}}}
	cache.Set("first", page)
	page.Options[0].Label = "changed"
	got, ok := cache.Get("first")
	if !ok || got.Options[0].Label != "One" {
		t.Fatalf("cache returned mutable source page: %#v, %v", got, ok)
	}
	cache.Set("second", LookupPage{Options: []LookupOption{{Value: "2", Label: "Two"}}})
	if _, ok := cache.Get("first"); ok {
		t.Fatal("cache exceeded its entry bound")
	}
	short := NewMemoryLookupCache(MemoryLookupCacheOptions{TTL: time.Nanosecond})
	short.Set("expired", page)
	time.Sleep(time.Millisecond)
	if _, ok := short.Get("expired"); ok {
		t.Fatal("expired cache entry returned")
	}
	cache.Clear()
	if _, ok := cache.Get("second"); ok {
		t.Fatal("Clear left an entry in the cache")
	}
}

func TestMemoryLookupCacheConcurrentAccess(t *testing.T) {
	cache := NewMemoryLookupCache(MemoryLookupCacheOptions{MaxEntries: 4})
	var group sync.WaitGroup
	for index := 0; index < 32; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			key := "key"
			cache.Set(key, LookupPage{Options: []LookupOption{{Value: index, Label: "value"}}})
			_, _ = cache.Get(key)
		}(index)
	}
	group.Wait()
}

func TestLookupCacheKeyIncludesServerOwnedFixedFilters(t *testing.T) {
	state := readState{definition: Definition{Scope: ScopeRequirements{Mode: ScopeModeGlobal}}}
	base := LookupQuery{Resource: "reference_values", ValueField: "id", LabelField: "name", Size: 25}
	states := base
	states.FixedFilters = []FixedLookupFilter{{Field: "kind", Values: []Value{"state"}}}
	statesAndTerritories := base
	statesAndTerritories.FixedFilters = []FixedLookupFilter{{Field: "kind", Values: []Value{"state", "territory"}}}
	if lookupCacheKey("contacts", "reference_id", state, states) == lookupCacheKey("contacts", "reference_id", state, statesAndTerritories) {
		t.Fatal("fixed filters share a lookup cache key")
	}
}

func TestGlobalDefinitionIsExplicitlyReadOnly(t *testing.T) {
	definition := validDefinition("global_categories")
	definition.Scope = ScopeRequirements{Mode: ScopeModeGlobal}
	definition.Permissions = Permissions{Read: "global.read"}
	definition.Delete = DeletePolicy{Mode: DeleteModeNone}
	definition.Concurrency = ConcurrencyPolicy{Mode: ConcurrencyNone}
	definition.UOW = nil
	definition.Source = fakeSource{capabilities: Capabilities{CapabilityOffsetPage: {}, CapabilityTotalCount: {}}}
	if err := ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("global read-only definition rejected: %v", err)
	}
	definition.Permissions.Create = "global.create"
	if err := ValidateDefinition(context.Background(), definition); err == nil {
		t.Fatal("global mutable definition accepted")
	}
}
