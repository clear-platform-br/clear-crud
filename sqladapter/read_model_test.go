package sqladapter

import (
	"context"
	"testing"

	crud "github.com/clear-platform-br/clear-crud"
	_ "modernc.org/sqlite"
)

func TestReadModelGroupedReportUsesFullScopedFilteredSet(t *testing.T) {
	database := newAutoDatabase(t, `
		CREATE TABLE report_rows (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, year INTEGER, month INTEGER, name TEXT NOT NULL, amount INTEGER);
		INSERT INTO report_rows (id, tenant_id, year, month, name, amount) VALUES
			(1, 'tenant-a', 2025, 11, 'A', 100),
			(2, 'tenant-a', 2025, 11, 'B', 250),
			(3, 'tenant-a', 2025, 11, 'C', 300),
			(4, 'tenant-a', 2025, 12, 'D', 400),
			(5, 'tenant-a', 2026, 1, 'E', 500),
			(6, 'tenant-a', 2026, 1, 'F', 600),
			(7, 'tenant-b', 2025, 11, 'Other tenant', 90000);
	`)
	config := ReadModelConfig{
		Query:    `SELECT id, tenant_id, year, month, name AS document_name, amount AS amount_cents FROM report_rows`,
		IDColumn: "id", ScopeColumns: map[string]Identifier{"tenant_id": "tenant_id"},
		Fields:     map[crud.FieldKey]Identifier{"year": "year", "month": "month", "document_name": "document_name", "amount_cents": "amount_cents"},
		Searchable: []crud.FieldKey{"document_name"},
	}
	source, err := NewReadModel(context.Background(), database, config)
	if err != nil {
		t.Fatal(err)
	}
	ranges, err := source.NumericFilterRanges(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, false, []crud.FieldKey{"year", "month", "amount_cents"})
	if err != nil || ranges["year"] != (crud.NumericFilterRange{Minimum: "2025", Maximum: "2026"}) || ranges["month"] != (crud.NumericFilterRange{Minimum: "1", Maximum: "12"}) || ranges["amount_cents"] != (crud.NumericFilterRange{Minimum: "100", Maximum: "600"}) {
		t.Fatalf("scoped read-model numeric ranges = %#v, err %v", ranges, err)
	}
	report := crud.ReportDefinition{
		Groups:   []crud.ReportGroup{{Field: "year", Direction: crud.SortAscending}, {Field: "month", Direction: crud.SortAscending}},
		Measures: []crud.ReportMeasure{{Key: "documents", Operation: crud.ReportCount}, {Key: "amount", Operation: crud.ReportSum, Field: "amount_cents"}},
	}
	query := crud.Query{Sort: []crud.Sort{{Field: "year", Direction: crud.SortAscending}, {Field: "month", Direction: crud.SortAscending}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 2}}
	page, err := source.ListReport(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, query, report)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 2 || page.Total == nil || *page.Total != 6 || page.Report == nil || len(page.Report.Subtotals) != 2 {
		t.Fatalf("grouped report page = %#v", page)
	}
	if page.Report.Subtotals[0].Level != 1 || page.Report.Subtotals[0].Measures[0].Value != int64(4) || page.Report.Subtotals[0].Measures[1].Value != int64(1050) {
		t.Fatalf("year subtotal = %#v", page.Report.Subtotals[0])
	}
	if page.Report.Subtotals[1].Level != 2 || page.Report.Subtotals[1].Measures[0].Value != int64(3) || page.Report.Subtotals[1].Measures[1].Value != int64(650) {
		t.Fatalf("month subtotal = %#v", page.Report.Subtotals[1])
	}

	query.Filters = []crud.Filter{{Field: "year", Operator: crud.FilterEqual, Value: int64(2026)}}
	filtered, err := source.ListReport(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, query, report)
	if err != nil || filtered.Total == nil || *filtered.Total != 2 || len(filtered.Report.Subtotals) != 2 {
		t.Fatalf("filtered grouped report = %#v, %v", filtered, err)
	}
	for _, subtotal := range filtered.Report.Subtotals {
		if subtotal.Measures[0].Value != int64(2) || subtotal.Measures[1].Value != int64(1100) {
			t.Fatalf("filtered aggregate = %#v", subtotal)
		}
	}

	emptyQuery := query
	emptyQuery.Filters = []crud.Filter{{Field: "year", Operator: crud.FilterEqual, Value: int64(2040)}}
	empty, err := source.ListReport(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, emptyQuery, report)
	if err != nil || empty.Report == nil || len(empty.Report.Subtotals) != 0 {
		t.Fatalf("empty grouped report = %#v, %v", empty, err)
	}
}

func TestReadModelGroupedReportRejectsInvalidRequestsAndPaths(t *testing.T) {
	database := newAutoDatabase(t, `
		CREATE TABLE report_rows (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, year INTEGER, amount INTEGER);
		INSERT INTO report_rows (id, tenant_id, year, amount) VALUES (1, 'tenant-a', 2025, 10);
	`)
	source, err := NewReadModel(context.Background(), database, ReadModelConfig{
		Query: `SELECT id, tenant_id, year, amount FROM report_rows`, IDColumn: "id",
		ScopeColumns: map[string]Identifier{"tenant_id": "tenant_id"},
		Fields:       map[crud.FieldKey]Identifier{"year": "year", "amount": "amount"},
	})
	if err != nil {
		t.Fatal(err)
	}
	query := crud.Query{Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}}
	validReport := crud.ReportDefinition{Groups: []crud.ReportGroup{{Field: "year", Direction: crud.SortAscending}}, Measures: []crud.ReportMeasure{{Key: "count", Operation: crud.ReportCount}}}
	for name, test := range map[string]struct {
		source *ReadModelSource
		query  crud.Query
		report crud.ReportDefinition
	}{
		"nil source":    {nil, query, validReport},
		"cursor":        {source, crud.Query{Page: crud.PageRequest{Mode: crud.PageModeCursor, Size: 10}}, validReport},
		"no groups":     {source, query, crud.ReportDefinition{Measures: validReport.Measures}},
		"bad group":     {source, query, crud.ReportDefinition{Groups: []crud.ReportGroup{{Field: "missing", Direction: crud.SortAscending}}, Measures: validReport.Measures}},
		"bad sum field": {source, query, crud.ReportDefinition{Groups: validReport.Groups, Measures: []crud.ReportMeasure{{Key: "sum", Operation: crud.ReportSum, Field: "missing"}}}},
		"bad operation": {source, query, crud.ReportDefinition{Groups: validReport.Groups, Measures: []crud.ReportMeasure{{Key: "unknown", Operation: "avg"}}}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := test.source.ListReport(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, test.query, test.report); err == nil {
				t.Fatal("invalid grouped report request accepted")
			}
		})
	}
	if _, err := source.ListReport(context.Background(), crud.Scope{"tenant_id": ""}, query, validReport); codeOfReadModel(err) != crud.ErrorForbidden {
		t.Fatalf("missing grouped report scope error = %v", err)
	}

	paths, err := visibleReportPaths([]crud.Record{{Fields: crud.Fields{"year": int64(2025)}}}, validReport, 1)
	if err != nil || len(paths) != 1 || paths[0][0] != int64(2025) {
		t.Fatalf("visible report paths = %#v, %v", paths, err)
	}
	if _, err := visibleReportPaths([]crud.Record{{Fields: crud.Fields{}}}, validReport, 1); err == nil {
		t.Fatal("missing group field accepted")
	}
	if _, _, err := source.reportPathPredicate(nil, [][]crud.Value{{int64(2025)}}); err == nil {
		t.Fatal("empty report path columns accepted")
	}
	if _, _, err := source.reportPathPredicate([]Identifier{"year"}, [][]crud.Value{{int64(2025), int64(2026)}}); err == nil {
		t.Fatal("report path width mismatch accepted")
	}
	if _, _, err := source.reportPathPredicate([]Identifier{"year"}, [][]crud.Value{{float64(2025)}}); err == nil {
		t.Fatal("unsupported report path value accepted")
	}
	predicate, arguments, err := source.reportPathPredicate([]Identifier{"year"}, [][]crud.Value{{nil}})
	if err != nil || predicate != "(\"clear_crud_read_model\".\"year\" IS NULL)" || len(arguments) != 0 {
		t.Fatalf("nullable report path = %q %#v, %v", predicate, arguments, err)
	}
	if !reportScalar("2025") || !reportScalar(true) || !reportScalar(int64(2025)) || reportScalar(float64(2025)) {
		t.Fatal("report scalar allowlist is incorrect")
	}
}

var _ crud.ReportDataSource = (*ReadModelSource)(nil)

func TestReadModelJoinsAndCalculatesReadOnlyFields(t *testing.T) {
	database := newAutoDatabase(t, `
        CREATE TABLE states (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
        CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL, state_id INTEGER NOT NULL);
        INSERT INTO states (id, name) VALUES (1, 'São Paulo'), (2, 'Paraná');
        INSERT INTO contacts (id, tenant_id, name, state_id) VALUES
            (1, 'tenant-a', 'Contato A', 1),
            (2, 'tenant-a', 'Contato B', 2),
            (3, 'tenant-b', 'Contato C', 1);
    `)
	source, err := NewReadModel(context.Background(), database, readModelTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	page, err := source.List(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.Query{
		Sort: []crud.Sort{{Field: "name", Direction: crud.SortAscending}},
		Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 2 || page.Total == nil || *page.Total != 2 {
		t.Fatalf("read model page = %#v", page)
	}
	if got := page.Records[0].Fields["state_name"]; got != "São Paulo" {
		t.Fatalf("joined state = %#v", got)
	}
	if got := page.Records[0].Fields["summary"]; got != "Contato A — São Paulo" {
		t.Fatalf("calculated summary = %#v", got)
	}
	if page.Records[0].Version != 1 {
		t.Fatalf("read model version = %d, want stable read version", page.Records[0].Version)
	}

	filtered, err := source.List(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.Query{
		Filters: []crud.Filter{{Field: "state_name", Operator: crud.FilterEqual, Value: "São Paulo"}},
		Page:    crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10},
	})
	if err != nil || len(filtered.Records) != 1 || filtered.Records[0].ID != "1" {
		t.Fatalf("joined filter = %#v, %v", filtered, err)
	}

	searched, err := source.List(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.Query{
		Search: "Paraná",
		Page:   crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10},
	})
	if err != nil || len(searched.Records) != 1 || searched.Records[0].ID != "2" {
		t.Fatalf("read model search = %#v, %v", searched, err)
	}

	if _, err := source.Get(context.Background(), crud.Scope{"tenant_id": "tenant-b"}, "1"); codeOfReadModel(err) != crud.ErrorNotFound {
		t.Fatalf("cross-scope read model Get error = %v", err)
	}
	if _, err := source.Update(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, "1", 1, crud.Mutation{}); codeOfReadModel(err) != crud.ErrorInvalidRequest {
		t.Fatalf("read model Update error = %v", err)
	}
	if _, err := source.Lookup(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.LookupQuery{Size: 10}); codeOfReadModel(err) != crud.ErrorInvalidRequest {
		t.Fatalf("read model Lookup error = %v", err)
	}
}

func TestRegisterReadModelCreatesReadOnlyDefinition(t *testing.T) {
	database := newAutoDatabase(t, `
        CREATE TABLE states (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
        CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL, state_id INTEGER NOT NULL);
    `)
	definition := crud.Definition{
		Contract:    crud.ContractDefinitionV1,
		Key:         "read_model_contacts",
		Labels:      crud.Labels{Title: "crud.read_model_contacts.title", Singular: "crud.read_model_contacts.singular"},
		Scope:       crud.ScopeRequirements{Mode: crud.ScopeModeTenant, Keys: []string{"tenant_id"}},
		Permissions: crud.Permissions{Read: "crud.read_model_contacts.read"},
		Fields: []crud.Field{
			{Key: "name", Label: "crud.read_model_contacts.name", Type: crud.FieldString, Visible: true},
			{Key: "state_name", Label: "crud.read_model_contacts.state_name", Type: crud.FieldString, Visible: true},
			{Key: "summary", Label: "crud.read_model_contacts.summary", Type: crud.FieldString, Visible: true},
		},
		Grid: crud.GridDefinition{
			Columns: []crud.FieldKey{"name", "state_name", "summary"}, DefaultSort: []crud.Sort{{Field: "name", Direction: crud.SortAscending}},
			Pagination: crud.PaginationDefinition{Mode: crud.PageModeOffset, DefaultSize: 10, AllowedSizes: []uint16{10, 25, 50, 100}, Total: true},
		},
		Presentation: crud.Presentation{Collection: crud.CollectionTable, Density: crud.DensityComfortable},
		Delete:       crud.DeletePolicy{Mode: crud.DeleteModeNone},
		Concurrency:  crud.ConcurrencyPolicy{Mode: crud.ConcurrencyNone},
	}
	config := readModelTestConfig()
	config.Fields = nil
	config.Searchable = nil
	config.ScopeColumns = nil
	registry := crud.NewRegistry()
	if err := RegisterReadModel(context.Background(), database, registry, definition, config); err != nil {
		t.Fatal(err)
	}
	registered, ok := registry.Get(definition.Key)
	if !ok || registered.Source == nil || registered.Permissions.Create != "" || registered.Delete.Mode != crud.DeleteModeNone {
		t.Fatalf("registered read model = %#v", registered)
	}
	for _, field := range registered.Fields {
		if !field.ReadOnly {
			t.Fatalf("registered read model field %q is writable", field.Key)
		}
	}
	if len(registered.Grid.Searchable) != 3 || len(registered.Grid.Sortable) != 3 {
		t.Fatalf("read model grid defaults = searchable %#v sortable %#v", registered.Grid.Searchable, registered.Grid.Sortable)
	}
}

func TestNewReadModelRejectsUnsafeOrIncompleteQueries(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL)`)
	config := readModelTestConfig()
	for name, query := range map[string]string{
		"empty":         "",
		"mutation":      "UPDATE contacts SET name = 'bad'",
		"multiple":      "SELECT id, tenant_id, name FROM contacts; SELECT 1",
		"missing alias": "SELECT id, tenant_id FROM contacts",
	} {
		t.Run(name, func(t *testing.T) {
			config.Query = query
			if _, err := NewReadModel(context.Background(), database, config); err == nil {
				t.Fatal("unsafe or incomplete query accepted")
			}
		})
	}
}

func TestReadModelRejectsMutationsAndInvalidDirectQueries(t *testing.T) {
	database := newAutoDatabase(t, `
        CREATE TABLE states (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
        CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL, state_id INTEGER NOT NULL);
    `)
	source, err := NewReadModel(context.Background(), database, readModelTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Create(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.Mutation{}); codeOfReadModel(err) != crud.ErrorInvalidRequest {
		t.Fatalf("read model Create error = %v", err)
	}
	if err := source.Delete(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, "1", 1, crud.DeleteModeHardDelete); codeOfReadModel(err) != crud.ErrorDeleteRestricted {
		t.Fatalf("read model Delete error = %v", err)
	}
	if _, err := source.Get(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, ""); codeOfReadModel(err) != crud.ErrorInvalidRequest {
		t.Fatalf("empty read model id error = %v", err)
	}
	query := crud.Query{Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}}
	query.Sort = []crud.Sort{{Field: "unknown", Direction: crud.SortAscending}}
	if _, err := source.List(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, query); codeOfReadModel(err) != crud.ErrorInvalidRequest {
		t.Fatalf("unknown read model sort error = %v", err)
	}
	query.Sort = nil
	query.Filters = []crud.Filter{{Field: "name", Operator: crud.FilterIn}}
	if _, err := source.List(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, query); codeOfReadModel(err) != crud.ErrorInvalidRequest {
		t.Fatalf("empty read model IN error = %v", err)
	}
	var nilSource *ReadModelSource
	if _, err := nilSource.Metadata(context.Background()); err == nil {
		t.Fatal("nil read model metadata unexpectedly succeeded")
	}
}

func TestRegisterReadModelRejectsMutableDefinitions(t *testing.T) {
	database := newAutoDatabase(t, `
        CREATE TABLE states (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
        CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL, state_id INTEGER NOT NULL);
    `)
	config := readModelTestConfig()
	registry := crud.NewRegistry()
	if err := RegisterReadModel(context.Background(), database, nil, crud.Definition{}, config); err == nil {
		t.Fatal("nil read model registry unexpectedly accepted")
	}
	validDefinition := crud.Definition{Key: "read_model_contacts"}
	source, err := NewReadModel(context.Background(), database, config)
	if err != nil {
		t.Fatal(err)
	}
	validDefinition.Source = source
	if err := RegisterReadModel(context.Background(), database, registry, validDefinition, config); err == nil {
		t.Fatal("read model definition with source unexpectedly accepted")
	}
	if err := RegisterReadModel(context.Background(), database, registry, crud.Definition{Key: "read_model_scope"}, ReadModelConfig{ScopeColumns: map[string]Identifier{"other": "tenant_id"}}); err == nil {
		t.Fatal("undeclared read model scope unexpectedly accepted")
	}
	for name, definition := range map[string]crud.Definition{
		"create":  {Key: "read_model_create", Permissions: crud.Permissions{Create: "create"}},
		"update":  {Key: "read_model_update", Permissions: crud.Permissions{Update: "update"}},
		"delete":  {Key: "read_model_delete", Permissions: crud.Permissions{Delete: "delete"}},
		"archive": {Key: "read_model_archive", Delete: crud.DeletePolicy{Mode: crud.DeleteModeArchive}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := RegisterReadModel(context.Background(), database, registry, definition, config); err == nil {
				t.Fatal("mutable read model definition unexpectedly accepted")
			}
		})
	}
}

func TestNewReadModelRejectsInvalidConfiguration(t *testing.T) {
	database := newAutoDatabase(t, `
        CREATE TABLE states (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
        CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL, state_id INTEGER NOT NULL);
    `)
	base := readModelTestConfig()
	if _, err := NewReadModel(context.Background(), nil, base); err == nil {
		t.Fatal("nil read model database unexpectedly accepted")
	}
	cases := map[string]func(*ReadModelConfig){
		"scope key":            func(config *ReadModelConfig) { config.ScopeColumns["bad-key"] = "tenant_id" },
		"scope alias":          func(config *ReadModelConfig) { config.ScopeColumns["tenant_id"] = "bad-alias" },
		"field key":            func(config *ReadModelConfig) { config.Fields["bad-key"] = "name" },
		"field alias":          func(config *ReadModelConfig) { config.Fields["name"] = "bad-alias" },
		"duplicate alias":      func(config *ReadModelConfig) { config.Fields["summary"] = "id" },
		"unknown search field": func(config *ReadModelConfig) { config.Searchable = append(config.Searchable, "missing") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			config := cloneReadModelTestConfig(base)
			mutate(&config)
			if _, err := NewReadModel(context.Background(), database, config); err == nil {
				t.Fatal("invalid read model configuration unexpectedly accepted")
			}
		})
	}
}

func TestReadModelRejectsInvalidListRequests(t *testing.T) {
	database := newAutoDatabase(t, `
        CREATE TABLE states (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
        CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL, state_id INTEGER NOT NULL);
    `)
	source, err := NewReadModel(context.Background(), database, readModelTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]crud.Query{
		"archived":                {IncludeArchived: true},
		"cursor mode":             {Page: crud.PageRequest{Mode: crud.PageModeCursor, Number: 1, Size: 10}},
		"missing page number":     {Page: crud.PageRequest{Mode: crud.PageModeOffset, Size: 10}},
		"missing page size":       {Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1}},
		"oversized page":          {Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 101}},
		"unknown filter operator": {Filters: []crud.Filter{{Field: "name", Operator: crud.FilterOperator("unknown"), Value: "x"}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}},
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := source.List(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, query); codeOfReadModel(err) != crud.ErrorInvalidRequest {
				t.Fatalf("invalid list request error = %v", err)
			}
		})
	}
	if _, err := source.List(context.Background(), crud.Scope{"tenant_id": ""}, crud.Query{Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}}); codeOfReadModel(err) != crud.ErrorForbidden {
		t.Fatalf("missing read model scope error = %v", err)
	}
	if _, err := source.Get(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, "1"); codeOfReadModel(err) != crud.ErrorNotFound {
		t.Fatalf("missing read model record error = %v", err)
	}
	noSearchSource, err := NewReadModel(context.Background(), database, ReadModelConfig{
		Query:        readModelTestConfig().Query,
		IDColumn:     "id",
		ScopeColumns: map[string]Identifier{"tenant_id": "tenant_id"},
		Fields:       readModelTestConfig().Fields,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := noSearchSource.List(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.Query{Search: "x", Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}}); codeOfReadModel(err) != crud.ErrorInvalidRequest {
		t.Fatalf("missing searchable fields error = %v", err)
	}
}

func cloneReadModelTestConfig(config ReadModelConfig) ReadModelConfig {
	clone := config
	clone.ScopeColumns = map[string]Identifier{}
	for key, value := range config.ScopeColumns {
		clone.ScopeColumns[key] = value
	}
	clone.Fields = map[crud.FieldKey]Identifier{}
	for key, value := range config.Fields {
		clone.Fields[key] = value
	}
	clone.Searchable = append([]crud.FieldKey(nil), config.Searchable...)
	return clone
}

func readModelTestConfig() ReadModelConfig {
	return ReadModelConfig{
		Query: `SELECT c.id AS id, c.tenant_id AS tenant_id, c.name AS name,
            s.name AS state_name, c.name || ' — ' || s.name AS summary
            FROM contacts AS c JOIN states AS s ON s.id = c.state_id`,
		IDColumn:     "id",
		ScopeColumns: map[string]Identifier{"tenant_id": "tenant_id"},
		Fields: map[crud.FieldKey]Identifier{
			"name":       "name",
			"state_name": "state_name",
			"summary":    "summary",
		},
		Searchable: []crud.FieldKey{"name", "state_name", "summary"},
	}
}

func codeOfReadModel(err error) crud.ErrorCode {
	publicError, ok := err.(*crud.Error)
	if !ok || publicError == nil {
		return ""
	}
	return publicError.Code
}
