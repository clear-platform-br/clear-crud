package crud

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	maxReportLevels   = 4
	maxReportMeasures = 8
)

// ReportDefinition declares ordered, server-owned grouping levels and measures.
// It contains presentation metadata only; adapters execute the aggregation.
type ReportDefinition struct {
	Groups   []ReportGroup
	Measures []ReportMeasure
}

// ReportGroup declares one ordered break level in a report.
type ReportGroup struct {
	Field     FieldKey
	Direction SortDirection
}

// ReportOperation identifies a supported aggregation operation.
type ReportOperation string

const (
	ReportCount ReportOperation = "count"
	ReportSum   ReportOperation = "sum"
)

// ReportMeasure declares one server-owned aggregate value.
type ReportMeasure struct {
	Key       string
	Label     MessageCode
	Operation ReportOperation
	Field     FieldKey `json:",omitempty"`
	// NumericDisplay affects only how the renderer presents this measure.
	NumericDisplay *NumericDisplay `json:",omitempty"`
}

// ReportSummary contains subtotals for grouping paths represented on a page.
// Each subtotal is calculated across the complete filtered and scoped result.
type ReportSummary struct {
	Subtotals []ReportSubtotal
}

// ReportSubtotal is the aggregate for one prefix of the declared group levels.
type ReportSubtotal struct {
	Level    uint8
	Values   []Value
	Measures []ReportMeasureValue
}

// ReportMeasureValue contains one measure result in declaration order.
type ReportMeasureValue struct {
	Key   string
	Value Value
}

// ReportDataSource is an optional DataSource capability for report pages.
// Implementations must apply the supplied scope, filters and search to both
// records and subtotals, and calculate subtotals over the full matching set.
type ReportDataSource interface {
	ListReport(context.Context, Scope, Query, ReportDefinition) (Page, error)
}

func cloneReportDefinition(report *ReportDefinition) *ReportDefinition {
	if report == nil {
		return nil
	}
	clone := *report
	clone.Groups = append([]ReportGroup(nil), report.Groups...)
	clone.Measures = append([]ReportMeasure(nil), report.Measures...)
	for index := range clone.Measures {
		clone.Measures[index].NumericDisplay = cloneNumericDisplay(report.Measures[index].NumericDisplay)
	}
	return &clone
}

func validateReport(report *ReportDefinition, fields map[FieldKey]Field, sortable map[FieldKey]struct{}, pagination PaginationDefinition) error {
	if report == nil {
		return nil
	}
	if len(report.Groups) == 0 || len(report.Groups) > maxReportLevels {
		return invalidDefinition("grid.report.groups", "must contain between 1 and 4 levels")
	}
	if len(report.Measures) == 0 || len(report.Measures) > maxReportMeasures {
		return invalidDefinition("grid.report.measures", "must contain between 1 and 8 measures")
	}
	if pagination.Mode != PageModeOffset {
		return invalidDefinition("grid.report.pagination", "currently requires offset pagination")
	}
	seenGroups := make(map[FieldKey]struct{}, len(report.Groups))
	for index, group := range report.Groups {
		field, ok := fields[group.Field]
		if !ok || !field.Visible || field.Sensitive || field.Type == FieldLookup {
			return invalidDefinition(fmt.Sprintf("grid.report.groups[%d].field", index), "must reference a visible, non-sensitive scalar field")
		}
		if _, ok := sortable[group.Field]; !ok {
			return invalidDefinition(fmt.Sprintf("grid.report.groups[%d].field", index), "must be sortable")
		}
		if group.Direction != SortAscending && group.Direction != SortDescending {
			return invalidDefinition(fmt.Sprintf("grid.report.groups[%d].direction", index), "must be asc or desc")
		}
		if _, exists := seenGroups[group.Field]; exists {
			return invalidDefinition("grid.report.groups", "must not contain duplicate fields")
		}
		seenGroups[group.Field] = struct{}{}
	}
	seenMeasures := make(map[string]struct{}, len(report.Measures))
	for index, measure := range report.Measures {
		path := fmt.Sprintf("grid.report.measures[%d]", index)
		if !validKey(measure.Key) || measure.Label == "" {
			return invalidDefinition(path, "requires a lowercase key and label")
		}
		if err := validateNumericDisplay(measure.NumericDisplay, path+".numericDisplay"); err != nil {
			return err
		}
		if _, exists := seenMeasures[measure.Key]; exists {
			return invalidDefinition("grid.report.measures", "must not contain duplicate keys")
		}
		seenMeasures[measure.Key] = struct{}{}
		switch measure.Operation {
		case ReportCount:
			if measure.Field != "" {
				return invalidDefinition(path+".field", "must be empty for count")
			}
		case ReportSum:
			field, ok := fields[measure.Field]
			if !ok || field.Sensitive || field.Type != FieldInteger {
				return invalidDefinition(path+".field", "must reference a non-sensitive integer field")
			}
		default:
			return invalidDefinition(path+".operation", "is unknown")
		}
	}
	return nil
}

func validReportPage(definition Definition, page Page) bool {
	report := definition.Grid.Report
	if report == nil || page.Report == nil || len(page.Records) > int(page.Size) || len(page.Report.Subtotals) > int(page.Size)*len(report.Groups) {
		return false
	}
	expected := make(map[string]struct{}, len(page.Records)*len(report.Groups))
	for _, record := range page.Records {
		values := make([]Value, 0, len(report.Groups))
		for level, group := range report.Groups {
			field, ok := record.Fields[group.Field]
			if !ok || !allowedValue(field) {
				return false
			}
			values = append(values, field)
			key, ok := reportPathKey(uint8(level+1), values)
			if !ok {
				return false
			}
			expected[key] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(page.Report.Subtotals))
	for _, subtotal := range page.Report.Subtotals {
		if subtotal.Level == 0 || int(subtotal.Level) > len(report.Groups) || len(subtotal.Values) != int(subtotal.Level) || len(subtotal.Measures) != len(report.Measures) {
			return false
		}
		key, ok := reportPathKey(subtotal.Level, subtotal.Values)
		if !ok {
			return false
		}
		if _, exists := expected[key]; !exists {
			return false
		}
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
		for index, measure := range report.Measures {
			value := subtotal.Measures[index]
			if value.Key != measure.Key || !validReportMeasureValue(measure, value.Value) {
				return false
			}
		}
	}
	return len(seen) == len(expected)
}

func reportPathKey(level uint8, values []Value) (string, bool) {
	for _, value := range values {
		if !allowedValue(value) {
			return "", false
		}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%d:%s", level, encoded), true
}

func validReportMeasureValue(measure ReportMeasure, value Value) bool {
	if value == nil {
		return measure.Operation == ReportSum
	}
	integer, ok := value.(int64)
	if !ok {
		return false
	}
	return measure.Operation != ReportCount || integer >= 0
}
