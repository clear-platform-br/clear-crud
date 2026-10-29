package crud

import (
	"context"
	"testing"
)

func TestValidateReportRejectsUnsafeOrAmbiguousDeclarations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*ReportDefinition, map[FieldKey]Field, map[FieldKey]struct{}, *PaginationDefinition)
		path   string
	}{
		{name: "no groups", mutate: func(report *ReportDefinition, _ map[FieldKey]Field, _ map[FieldKey]struct{}, _ *PaginationDefinition) {
			report.Groups = nil
		}, path: "grid.report.groups"},
		{name: "group field hidden", mutate: func(report *ReportDefinition, fields map[FieldKey]Field, _ map[FieldKey]struct{}, _ *PaginationDefinition) {
			fields["year"] = Field{Key: "year", Visible: false}
		}, path: "grid.report.groups[0].field"},
		{name: "group field non sortable", mutate: func(_ *ReportDefinition, _ map[FieldKey]Field, sortable map[FieldKey]struct{}, _ *PaginationDefinition) {
			delete(sortable, "year")
		}, path: "grid.report.groups[0].field"},
		{name: "group field lookup", mutate: func(_ *ReportDefinition, fields map[FieldKey]Field, _ map[FieldKey]struct{}, _ *PaginationDefinition) {
			fields["year"] = Field{Key: "year", Visible: true, Type: FieldLookup}
		}, path: "grid.report.groups[0].field"},
		{name: "duplicate group", mutate: func(report *ReportDefinition, _ map[FieldKey]Field, _ map[FieldKey]struct{}, _ *PaginationDefinition) {
			report.Groups = append(report.Groups, report.Groups[0])
		}, path: "grid.report.groups"},
		{name: "unknown operation", mutate: func(report *ReportDefinition, _ map[FieldKey]Field, _ map[FieldKey]struct{}, _ *PaginationDefinition) {
			report.Measures[0].Operation = "avg"
		}, path: "grid.report.measures[0].operation"},
		{name: "sum requires integer", mutate: func(report *ReportDefinition, fields map[FieldKey]Field, _ map[FieldKey]struct{}, _ *PaginationDefinition) {
			report.Measures[0] = ReportMeasure{Key: "sum", Label: "crud.sum", Operation: ReportSum, Field: "amount"}
			fields["amount"] = Field{Key: "amount", Type: FieldString}
		}, path: "grid.report.measures[0].field"},
		{name: "cursor pagination", mutate: func(_ *ReportDefinition, _ map[FieldKey]Field, _ map[FieldKey]struct{}, page *PaginationDefinition) {
			page.Mode = PageModeCursor
		}, path: "grid.report.pagination"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			report := &ReportDefinition{
				Groups:   []ReportGroup{{Field: "year", Direction: SortAscending}},
				Measures: []ReportMeasure{{Key: "count", Label: "crud.count", Operation: ReportCount}},
			}
			fields := map[FieldKey]Field{
				"year":   {Key: "year", Visible: true, Type: FieldInteger},
				"amount": {Key: "amount", Type: FieldInteger},
			}
			sortable := map[FieldKey]struct{}{"year": {}}
			pagination := PaginationDefinition{Mode: PageModeOffset}
			test.mutate(report, fields, sortable, &pagination)
			if err := validateReport(report, fields, sortable, pagination); err == nil {
				t.Fatal("validateReport() accepted invalid declaration")
			} else if definitionError, ok := err.(*DefinitionError); !ok || definitionError.Path != test.path {
				t.Fatalf("validateReport() error = %v, want path %q", err, test.path)
			}
		})
	}
}

func TestValidReportPageRequiresEveryPageGroupSubtotalAndMeasure(t *testing.T) {
	t.Parallel()
	definition := Definition{Grid: GridDefinition{Report: &ReportDefinition{
		Groups:   []ReportGroup{{Field: "year", Direction: SortAscending}, {Field: "month", Direction: SortAscending}},
		Measures: []ReportMeasure{{Key: "count", Label: "crud.count", Operation: ReportCount}, {Key: "amount", Label: "crud.amount", Operation: ReportSum, Field: "amount"}},
	}}}
	page := Page{Size: 2, Records: []Record{
		{Fields: Fields{"year": int64(2025), "month": int64(11)}},
		{Fields: Fields{"year": int64(2025), "month": int64(12)}},
	}, Report: &ReportSummary{Subtotals: []ReportSubtotal{
		{Level: 1, Values: []Value{int64(2025)}, Measures: []ReportMeasureValue{{Key: "count", Value: int64(2)}, {Key: "amount", Value: int64(100)}}},
		{Level: 2, Values: []Value{int64(2025), int64(11)}, Measures: []ReportMeasureValue{{Key: "count", Value: int64(1)}, {Key: "amount", Value: int64(40)}}},
		{Level: 2, Values: []Value{int64(2025), int64(12)}, Measures: []ReportMeasureValue{{Key: "count", Value: int64(1)}, {Key: "amount", Value: int64(60)}}},
	}}}
	if !validReportPage(definition, page) {
		t.Fatal("valid report page rejected")
	}
	page.Report.Subtotals[0].Measures[1].Key = "unexpected"
	if validReportPage(definition, page) {
		t.Fatal("invalid report measure key accepted")
	}
	page.Report.Subtotals[0].Measures[1].Key = "amount"
	page.Report.Subtotals[0].Measures[1].Value = nil
	if !validReportPage(definition, page) {
		t.Fatal("nullable sum result rejected")
	}
	page.Report.Subtotals[0].Measures[0].Value = nil
	if validReportPage(definition, page) {
		t.Fatal("nullable count result accepted")
	}
	page.Report.Subtotals = page.Report.Subtotals[:1]
	if validReportPage(definition, page) {
		t.Fatal("missing page subtotal accepted")
	}
	page.Report.Subtotals = append(page.Report.Subtotals, ReportSubtotal{Level: 3, Values: []Value{int64(2025), int64(11), "extra"}})
	if validReportPage(definition, page) {
		t.Fatal("unknown grouping level accepted")
	}
}

func TestReportSortKeepsDeclaredGroupsFirstAndDeduplicates(t *testing.T) {
	t.Parallel()
	got := reportSort(ReportDefinition{Groups: []ReportGroup{
		{Field: "year", Direction: SortAscending}, {Field: "month", Direction: SortDescending},
	}}, []Sort{{Field: "name", Direction: SortAscending}, {Field: "year", Direction: SortDescending}})
	want := []Sort{{Field: "year", Direction: SortAscending}, {Field: "month", Direction: SortDescending}, {Field: "name", Direction: SortAscending}}
	if len(got) != len(want) {
		t.Fatalf("report sort = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("report sort = %#v, want %#v", got, want)
		}
	}
}

func TestReportMeasureValuesAreBoundedIntegers(t *testing.T) {
	t.Parallel()
	count := ReportMeasure{Operation: ReportCount}
	sum := ReportMeasure{Operation: ReportSum}
	if !validReportMeasureValue(count, int64(0)) || validReportMeasureValue(count, int64(-1)) {
		t.Fatal("count must be a non-negative integer")
	}
	if !validReportMeasureValue(sum, int64(1)) || !validReportMeasureValue(sum, nil) || validReportMeasureValue(sum, 1.5) {
		t.Fatal("sum must be an integer or null")
	}
}

func TestNumericFilterRangesAreLimitedToVisibleNumericGridFields(t *testing.T) {
	t.Parallel()
	definition := Definition{
		Fields: []Field{
			{Key: "year", Type: FieldInteger, Visible: true},
			{Key: "amount", Type: FieldDecimal, Visible: true},
			{Key: "secret", Type: FieldInteger, Visible: true, Sensitive: true},
			{Key: "name", Type: FieldString, Visible: true},
		},
		Grid: GridDefinition{Columns: []FieldKey{"year", "amount", "secret", "name"}},
	}
	fields := numericGridFields(definition)
	if len(fields) != 2 || fields[0] != "year" || fields[1] != "amount" {
		t.Fatalf("numeric filter fields = %#v", fields)
	}
	valid := map[FieldKey]NumericFilterRange{
		"year":   {Minimum: "1969", Maximum: "2003"},
		"amount": {Minimum: "100.00", Maximum: "1000000.00"},
	}
	if !validNumericFilterRanges(definition, fields, valid) {
		t.Fatal("valid numeric bounds rejected")
	}
	for name, ranges := range map[string]map[FieldKey]NumericFilterRange{
		"out-of-order":    {"year": {Minimum: "2003", Maximum: "1969"}},
		"sensitive-field": {"secret": {Minimum: "1", Maximum: "2"}},
		"invalid-integer": {"year": {Minimum: "1.5", Maximum: "2"}},
		"invalid-decimal": {"amount": {Minimum: "not-a-number", Maximum: "2"}},
		"empty-bound":     {"year": {Minimum: "", Maximum: "2"}},
		"unknown-field":   {"unknown": {Minimum: "1", Maximum: "2"}},
	} {
		t.Run(name, func(t *testing.T) {
			if validNumericFilterRanges(definition, fields, ranges) {
				t.Fatal("invalid numeric bounds accepted")
			}
		})
	}
	if validNumericFilterRanges(definition, []FieldKey{"unknown"}, nil) {
		t.Fatal("missing field was accepted as a numeric filter field")
	}
	if validNumericFilterRanges(definition, []FieldKey{"name"}, nil) {
		t.Fatal("non-numeric field was accepted as a numeric filter field")
	}
}

type reportTestSource struct{ fakeSource }

func (source reportTestSource) ListReport(context.Context, Scope, Query, ReportDefinition) (Page, error) {
	return Page{}, nil
}

func TestReportDefinitionRequiresAdapterCapability(t *testing.T) {
	t.Parallel()
	definition := validDefinition("report_test")
	definition.Fields = append(definition.Fields, Field{Key: "year", Label: "crud.year", Type: FieldInteger, Visible: true})
	definition.Grid.Columns = append(definition.Grid.Columns, "year")
	definition.Grid.Sortable = append(definition.Grid.Sortable, "year")
	definition.Grid.Report = &ReportDefinition{
		Groups:   []ReportGroup{{Field: "year", Direction: SortAscending}},
		Measures: []ReportMeasure{{Key: "count", Label: "crud.count", Operation: ReportCount}},
	}
	if err := ValidateDefinition(context.Background(), definition); err == nil {
		t.Fatal("report source without ReportDataSource accepted")
	}

	definition.Source = reportTestSource{fakeSource{capabilities: mutableCapabilities()}}
	if err := ValidateDefinition(context.Background(), definition); err == nil {
		t.Fatal("report source without grouped-report capability accepted")
	}

	capabilities := mutableCapabilities()
	capabilities[CapabilityGroupedReport] = struct{}{}
	definition.Source = reportTestSource{fakeSource{capabilities: capabilities}}
	if err := ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("valid report source rejected: %v", err)
	}
}

func TestCloneReportDefinitionCopiesSlices(t *testing.T) {
	t.Parallel()
	original := &ReportDefinition{
		Groups:   []ReportGroup{{Field: "year", Direction: SortAscending}},
		Measures: []ReportMeasure{{Key: "count", Label: "crud.count", Operation: ReportCount}},
	}
	clone := cloneReportDefinition(original)
	clone.Groups[0].Field = "month"
	clone.Measures[0].Key = "other"
	if original.Groups[0].Field != "year" || original.Measures[0].Key != "count" {
		t.Fatal("report clone shares declaration slices with original")
	}
}
