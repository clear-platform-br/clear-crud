package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	crud "github.com/clear-platform-br/clear-crud"
	_ "modernc.org/sqlite"
)

const demoTable = "clear_crud_minimal_demo_contacts"
const demoTranslatedTable = "clear_crud_minimal_demo_contacts_ptbr"
const demoDefaultsTable = "clear_crud_minimal_demo_contacts_defaults"
const demoPatternTable = "clear_crud_minimal_demo_contacts_pattern"
const demoOrderedTable = "clear_crud_minimal_demo_contacts_ordered"
const demoEnumControlsTable = "clear_crud_enum_controls_demo"
const demoEnumTable = "clear_crud_disposable_junk_contacts_enum"
const demoLookupTable = "clear_crud_reference_lookup_demo"
const demoReadModelTable = "clear_crud_reference_read_model_demo"
const demoAuxiliaryCatalogTable = "clear_crud_auxiliary_catalogs"
const demoAuxiliaryCatalogLookupTable = "clear_crud_auxiliary_catalog_lookup_demo"
const demoAuxiliaryCatalogChannelLookupTable = "clear_crud_auxiliary_catalog_channel_lookup_demo"

func TestDemoHandlerServesTheAutomaticResource(t *testing.T) {
	database := newDemoDatabase(t)
	handler, err := newHandler(database)
	if err != nil {
		t.Fatal(err)
	}
	definitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoTable+"/definition", nil)
	definitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(definitionResponse, definitionRequest)
	if definitionResponse.Code != http.StatusOK {
		t.Fatalf("%s %s = %d: %s", definitionRequest.Method, definitionRequest.URL, definitionResponse.Code, definitionResponse.Body.String())
	}
	var envelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(definitionResponse.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if got := len(envelope.Data.Fields); got != 4 {
		t.Fatalf("definition fields = %d, want 4", got)
	}
	if got := len(envelope.Data.Grid.Columns); got != 4 {
		t.Fatalf("definition list columns = %d, want 4", got)
	}
	if got := envelope.Data.Grid.Pagination.DefaultSize; got != 25 {
		t.Fatalf("definition default page size = %d, want 25", got)
	}
	if got := envelope.Data.Form.Fields; len(got) != 0 {
		t.Fatalf("definition form fields = %#v, want automatic visible fields", got)
	}
	if status := findPublicField(t, envelope.Data.Fields, "status"); status.Default != "new" {
		t.Fatalf("status default = %#v, want schema default new", status.Default)
	}
	if enabled := findPublicField(t, envelope.Data.Fields, "enabled"); enabled.Default != true {
		t.Fatalf("enabled default = %#v, want schema default true", enabled.Default)
	}
	if got := envelope.Data.Grid.DefaultSort; len(got) != 2 || got[0].Field != "enabled" || got[1].Field != "name" {
		t.Fatalf("definition default sort = %#v, want enabled then name", got)
	}
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoTable+"/records?page=1&size=25", nil),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s %s = %d: %s", request.Method, request.URL, response.Code, response.Body.String())
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoTable+"/records", strings.NewReader(`{"fields":{"name":"Demo record","notes":"Minimal table-only definition","status":"new","enabled":true}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoTable+"/records", strings.NewReader(`{"fields":{"name":"Schema defaults record","notes":"defaults smoke"}}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create with omitted defaults = %d: %s", response.Code, response.Body.String())
	}
	var defaultedRecordEnvelope struct {
		Data crud.Record `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &defaultedRecordEnvelope); err != nil {
		t.Fatal(err)
	}
	if defaultedRecordEnvelope.Data.Fields["status"] != "new" || defaultedRecordEnvelope.Data.Fields["enabled"] != float64(1) {
		t.Fatalf("schema defaults on create = %#v", defaultedRecordEnvelope.Data.Fields)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoTable+"/records", strings.NewReader(`{"fields":{"name":"`+strings.Repeat("a", 256)+`","notes":"too long","status":"new","enabled":true}}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("oversized name = %d: %s", response.Code, response.Body.String())
	}
	translatedDefinitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoTranslatedTable+"/definition", nil)
	translatedDefinitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(translatedDefinitionResponse, translatedDefinitionRequest)
	if translatedDefinitionResponse.Code != http.StatusOK {
		t.Fatalf("translated definition = %d: %s", translatedDefinitionResponse.Code, translatedDefinitionResponse.Body.String())
	}
	var translatedEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(translatedDefinitionResponse.Body.Bytes(), &translatedEnvelope); err != nil {
		t.Fatal(err)
	}
	translatedStatus := findPublicField(t, translatedEnvelope.Data.Fields, "status")
	if translatedEnvelope.Data.Grid.Pagination.DefaultSize != 10 || translatedStatus.Type != crud.FieldEnum || len(translatedStatus.Enum) != 3 {
		t.Fatalf("translated definition = %#v, pagination = %#v", translatedStatus, translatedEnvelope.Data.Grid.Pagination)
	}
	if translatedStatus.Enum[0].Label != "crud.status.new" || translatedStatus.Enum[1].Label != "crud.status.review" || translatedStatus.Enum[2].Label != "crud.status.closed" {
		t.Fatalf("translated enum labels = %#v", translatedStatus.Enum)
	}
	controlsDefinitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoEnumControlsTable+"/definition", nil)
	controlsDefinitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(controlsDefinitionResponse, controlsDefinitionRequest)
	if controlsDefinitionResponse.Code != http.StatusOK {
		t.Fatalf("enum controls definition = %d: %s", controlsDefinitionResponse.Code, controlsDefinitionResponse.Body.String())
	}
	var controlsEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(controlsDefinitionResponse.Body.Bytes(), &controlsEnvelope); err != nil {
		t.Fatal(err)
	}
	for fieldKey, expectedControl := range map[crud.FieldKey]crud.EnumControl{
		"priority": crud.EnumControlSelect,
		"channel":  crud.EnumControlRadio,
		"state":    crud.EnumControlSegmented,
		"tone":     crud.EnumControlButtons,
	} {
		field := findPublicField(t, controlsEnvelope.Data.Fields, fieldKey)
		if field.Type != crud.FieldEnum || field.EnumControl != expectedControl || len(field.Enum) != 3 {
			t.Fatalf("enum control field %q = %#v, want %q and three options", fieldKey, field, expectedControl)
		}
	}
	gotColumns := controlsEnvelope.Data.Grid.Columns
	if controlsEnvelope.Data.Grid.Pagination.DefaultSize != 10 || len(gotColumns) != 5 || gotColumns[0] != "name" || gotColumns[1] != "priority" || gotColumns[2] != "channel" || gotColumns[3] != "state" || gotColumns[4] != "tone" {
		t.Fatalf("enum controls grid = %#v, pagination = %#v", controlsEnvelope.Data.Grid.Columns, controlsEnvelope.Data.Grid.Pagination)
	}
	if controlsEnvelope.Data.Delete.Mode != crud.DeleteModeHardDelete || !containsAction(controlsEnvelope.Data.Actions, crud.ActionDelete) {
		t.Fatalf("enum controls delete policy = %#v, actions = %#v; want explicit hard delete", controlsEnvelope.Data.Delete, controlsEnvelope.Data.Actions)
	}
	enumDefinitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoEnumTable+"/definition", nil)
	enumDefinitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(enumDefinitionResponse, enumDefinitionRequest)
	if enumDefinitionResponse.Code != http.StatusOK {
		t.Fatalf("enum definition = %d: %s", enumDefinitionResponse.Code, enumDefinitionResponse.Body.String())
	}
	var enumEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(enumDefinitionResponse.Body.Bytes(), &enumEnvelope); err != nil {
		t.Fatal(err)
	}
	status := findPublicField(t, enumEnvelope.Data.Fields, "status")
	if status.Type != crud.FieldEnum || len(status.Enum) != 3 || enumEnvelope.Data.Grid.Columns[1] != "status" {
		t.Fatalf("enum definition = %#v, grid = %#v", status, enumEnvelope.Data.Grid.Columns)
	}
	if enumEnvelope.Data.Grid.ArchiveVisibility != crud.ArchiveVisibilityActiveAndArchived {
		t.Fatalf("enum archive visibility = %q, want active_and_archived", enumEnvelope.Data.Grid.ArchiveVisibility)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoEnumTable+"/records", strings.NewReader(`{"fields":{"name":"Enum demo","status":"not-allowed","enabled":true,"notes":"demo"}}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid enum = %d: %s", response.Code, response.Body.String())
	}
	lookupDefinitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoLookupTable+"/definition", nil)
	lookupDefinitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(lookupDefinitionResponse, lookupDefinitionRequest)
	if lookupDefinitionResponse.Code != http.StatusOK {
		t.Fatalf("lookup definition = %d: %s", lookupDefinitionResponse.Code, lookupDefinitionResponse.Body.String())
	}
	var lookupEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(lookupDefinitionResponse.Body.Bytes(), &lookupEnvelope); err != nil {
		t.Fatal(err)
	}
	stateField := findPublicField(t, lookupEnvelope.Data.Fields, "state_id")
	regionField := findPublicField(t, lookupEnvelope.Data.Fields, "region_id")
	cityField := findPublicField(t, lookupEnvelope.Data.Fields, "city_id")
	if stateField.Type != crud.FieldLookup || stateField.Lookup == nil || stateField.Lookup.Resource != "clear_crud_reference_values" || len(stateField.Lookup.FixedFilters) != 0 || regionField.Lookup == nil || len(regionField.Lookup.Dependencies) != 1 || cityField.Lookup == nil || cityField.Lookup.Dependencies[0] != "region_id" {
		t.Fatalf("lookup chain definition = %#v %#v %#v", stateField, regionField, cityField)
	}
	rootLookupRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoLookupTable+"/lookups/state_id", nil)
	rootLookupResponse := httptest.NewRecorder()
	handler.ServeHTTP(rootLookupResponse, rootLookupRequest)
	if rootLookupResponse.Code != http.StatusOK {
		t.Fatalf("root lookup = %d: %s", rootLookupResponse.Code, rootLookupResponse.Body.String())
	}
	auxiliaryDefinitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoAuxiliaryCatalogTable+"/definition", nil)
	auxiliaryDefinitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(auxiliaryDefinitionResponse, auxiliaryDefinitionRequest)
	if auxiliaryDefinitionResponse.Code != http.StatusOK {
		t.Fatalf("auxiliary catalog definition = %d: %s", auxiliaryDefinitionResponse.Code, auxiliaryDefinitionResponse.Body.String())
	}
	var auxiliaryEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(auxiliaryDefinitionResponse.Body.Bytes(), &auxiliaryEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(auxiliaryEnvelope.Data.Details) != 1 || auxiliaryEnvelope.Data.Details[0].Key != "options" {
		t.Fatalf("auxiliary catalog details = %#v", auxiliaryEnvelope.Data.Details)
	}
	if len(auxiliaryEnvelope.Data.Grid.Columns) != 1 || auxiliaryEnvelope.Data.Grid.Columns[0] != "title" {
		t.Fatalf("auxiliary catalog grid = %#v, want only table title", auxiliaryEnvelope.Data.Grid.Columns)
	}
	if auxiliaryEnvelope.Data.Presentation.TitleField != "title" {
		t.Fatalf("auxiliary catalog title field = %q, want title", auxiliaryEnvelope.Data.Presentation.TitleField)
	}
	auxiliaryLookupDefinitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoAuxiliaryCatalogLookupTable+"/definition", nil)
	auxiliaryLookupDefinitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(auxiliaryLookupDefinitionResponse, auxiliaryLookupDefinitionRequest)
	if auxiliaryLookupDefinitionResponse.Code != http.StatusOK {
		t.Fatalf("auxiliary catalog lookup definition = %d: %s", auxiliaryLookupDefinitionResponse.Code, auxiliaryLookupDefinitionResponse.Body.String())
	}
	var auxiliaryLookupEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(auxiliaryLookupDefinitionResponse.Body.Bytes(), &auxiliaryLookupEnvelope); err != nil {
		t.Fatal(err)
	}
	selectedOption := findPublicField(t, auxiliaryLookupEnvelope.Data.Fields, "selected_option")
	if selectedOption.Lookup == nil || selectedOption.Lookup.Resource != "clear_crud_auxiliary_catalog_options" || len(selectedOption.Lookup.FixedFilters) != 0 {
		t.Fatalf("auxiliary catalog lookup definition = %#v", selectedOption)
	}
	optionsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoAuxiliaryCatalogLookupTable+"/lookups/selected_option", nil)
	optionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(optionsResponse, optionsRequest)
	if optionsResponse.Code != http.StatusOK || !strings.Contains(optionsResponse.Body.String(), "Fernando de Noronha") || strings.Contains(optionsResponse.Body.String(), "Território inativo") {
		t.Fatalf("auxiliary catalog lookup = %d: %s", optionsResponse.Code, optionsResponse.Body.String())
	}
	channelDefinitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoAuxiliaryCatalogChannelLookupTable+"/definition", nil)
	channelDefinitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(channelDefinitionResponse, channelDefinitionRequest)
	if channelDefinitionResponse.Code != http.StatusOK {
		t.Fatalf("auxiliary catalog channel definition = %d: %s", channelDefinitionResponse.Code, channelDefinitionResponse.Body.String())
	}
	var channelEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(channelDefinitionResponse.Body.Bytes(), &channelEnvelope); err != nil {
		t.Fatal(err)
	}
	channelField := findPublicField(t, channelEnvelope.Data.Fields, "channel")
	if channelField.Lookup == nil || channelField.Lookup.Resource != "clear_crud_auxiliary_catalog_options" || len(channelField.Lookup.FixedFilters) != 0 {
		t.Fatalf("auxiliary catalog channel lookup definition = %#v", channelField)
	}
	channelLookupRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoAuxiliaryCatalogChannelLookupTable+"/lookups/channel", nil)
	channelLookupResponse := httptest.NewRecorder()
	handler.ServeHTTP(channelLookupResponse, channelLookupRequest)
	if channelLookupResponse.Code != http.StatusOK || !strings.Contains(channelLookupResponse.Body.String(), "E-mail") || !strings.Contains(channelLookupResponse.Body.String(), "Telefone") || !strings.Contains(channelLookupResponse.Body.String(), "WhatsApp") || strings.Contains(channelLookupResponse.Body.String(), "Baixa") || strings.Contains(channelLookupResponse.Body.String(), "Alta") || strings.Contains(channelLookupResponse.Body.String(), "São Paulo") || strings.Contains(channelLookupResponse.Body.String(), "Fernando de Noronha") || strings.Contains(channelLookupResponse.Body.String(), "Canal inativo") {
		t.Fatalf("auxiliary catalog channel lookup leaked another catalog or inactive data: %d: %s", channelLookupResponse.Code, channelLookupResponse.Body.String())
	}
	var channelLookupEnvelope struct {
		Data []crud.LookupOption `json:"data"`
	}
	if err := json.Unmarshal(channelLookupResponse.Body.Bytes(), &channelLookupEnvelope); err != nil {
		t.Fatal(err)
	}
	wantChannelValues := []crud.Value{"email", "phone", "whatsapp"}
	if len(channelLookupEnvelope.Data) != len(wantChannelValues) {
		t.Fatalf("channel lookup option count = %d, want %d", len(channelLookupEnvelope.Data), len(wantChannelValues))
	}
	for index, want := range wantChannelValues {
		if channelLookupEnvelope.Data[index].Value != want {
			t.Fatalf("channel lookup option %d = %#v, want value %q", index, channelLookupEnvelope.Data[index], want)
		}
	}
	var rootLookupEnvelope struct {
		Data []crud.LookupOption `json:"data"`
	}
	if err := json.Unmarshal(rootLookupResponse.Body.Bytes(), &rootLookupEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(rootLookupEnvelope.Data) != 25 {
		t.Fatalf("root lookup page size = %d, want 25", len(rootLookupEnvelope.Data))
	}
	for _, option := range rootLookupEnvelope.Data {
		if option.Label == "Real" {
			t.Fatalf("fixed lookup filter leaked another catalog: %#v", rootLookupEnvelope.Data)
		}
	}
	currencyLookupRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoLookupTable+"/lookups/state_id?q=Real", nil)
	currencyLookupResponse := httptest.NewRecorder()
	handler.ServeHTTP(currencyLookupResponse, currencyLookupRequest)
	if currencyLookupResponse.Code != http.StatusOK || strings.Contains(currencyLookupResponse.Body.String(), "Real") {
		t.Fatalf("fixed lookup filter accepted currency: %d: %s", currencyLookupResponse.Code, currencyLookupResponse.Body.String())
	}
	searchLookupRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoLookupTable+"/lookups/state_id?q=Tocantins", nil)
	searchLookupResponse := httptest.NewRecorder()
	handler.ServeHTTP(searchLookupResponse, searchLookupRequest)
	if searchLookupResponse.Code != http.StatusOK || !strings.Contains(searchLookupResponse.Body.String(), "Tocantins") {
		t.Fatalf("lookup search beyond first page = %d: %s", searchLookupResponse.Code, searchLookupResponse.Body.String())
	}
	lookupRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoLookupTable+"/lookups/region_id?depends.state_id=1", nil)
	lookupResponse := httptest.NewRecorder()
	handler.ServeHTTP(lookupResponse, lookupRequest)
	if lookupResponse.Code != http.StatusOK || !strings.Contains(lookupResponse.Body.String(), "Região Metropolitana de São Paulo") || strings.Contains(lookupResponse.Body.String(), "Região Metropolitana de Curitiba") {
		t.Fatalf("dependent lookup = %d: %s", lookupResponse.Code, lookupResponse.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoLookupTable+"/records", strings.NewReader(`{"fields":{"name":"Contato novo","state_id":1,"region_id":11,"city_id":111}}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("lookup create with numeric IDs = %d: %s", response.Code, response.Body.String())
	}
}

func TestDemoPatternSmokeValidatesServerOwnedFormat(t *testing.T) {
	database := newDemoDatabase(t)
	handler, err := newHandler(database)
	if err != nil {
		t.Fatal(err)
	}
	definitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoPatternTable+"/definition", nil)
	definitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(definitionResponse, definitionRequest)
	if definitionResponse.Code != http.StatusOK {
		t.Fatalf("pattern definition = %d: %s", definitionResponse.Code, definitionResponse.Body.String())
	}
	var definitionEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(definitionResponse.Body.Bytes(), &definitionEnvelope); err != nil {
		t.Fatal(err)
	}
	field := findPublicField(t, definitionEnvelope.Data.Fields, "name")
	if field.Pattern != crud.PatternFirstLetterUpper || field.PatternMessage != "crud.validation.name_uppercase" {
		t.Fatalf("pattern metadata = %#v", field)
	}

	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoPatternTable+"/records", strings.NewReader(`{"fields":{"name":"lowercase name","notes":"pattern smoke","status":"new","enabled":true}}`))
	invalidRequest.Header.Set("Content-Type", "application/json")
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid pattern create = %d: %s", invalidResponse.Code, invalidResponse.Body.String())
	}
	if !strings.Contains(invalidResponse.Body.String(), "crud.validation.name_uppercase") {
		t.Fatalf("invalid pattern response = %s", invalidResponse.Body.String())
	}

	validRequest := httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoPatternTable+"/records", strings.NewReader(`{"fields":{"name":"Uppercase name","notes":"pattern smoke","status":"new","enabled":true}}`))
	validRequest.Header.Set("Content-Type", "application/json")
	validResponse := httptest.NewRecorder()
	handler.ServeHTTP(validResponse, validRequest)
	if validResponse.Code != http.StatusCreated {
		t.Fatalf("valid pattern create = %d: %s", validResponse.Code, validResponse.Body.String())
	}
}

func TestDemoHandlerSearchesEnumValues(t *testing.T) {
	database := newDemoDatabase(t)
	if _, err := database.Exec(`INSERT INTO clear_crud_disposable_junk_contacts (tenant_id, name, status, enabled, version) VALUES
		('demo_tenant_alpha', 'Novo', 'new', 1, 1),
		('demo_tenant_alpha', 'Em análise', 'review', 1, 1),
		('demo_tenant_alpha', 'Encerrado', 'closed', 1, 1)`); err != nil {
		t.Fatal(err)
	}
	handler, err := newHandler(database)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoEnumTable+"/records?page=1&size=10&q=review", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("enum search = %d: %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data []crud.Record `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) == 0 {
		t.Fatal("enum search returned no review records")
	}
	for _, record := range envelope.Data {
		if record.Fields["status"] != "review" {
			t.Fatalf("enum search returned status %#v in record %#v", record.Fields["status"], record.ID)
		}
	}
}

func TestDemoReadModelCombinesJoinedAndCalculatedFields(t *testing.T) {
	database := newDemoDatabase(t)
	handler, err := newHandler(database)
	if err != nil {
		t.Fatal(err)
	}

	definitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoReadModelTable+"/definition", nil)
	definitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(definitionResponse, definitionRequest)
	if definitionResponse.Code != http.StatusOK {
		t.Fatalf("read model definition = %d: %s", definitionResponse.Code, definitionResponse.Body.String())
	}
	var definitionEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(definitionResponse.Body.Bytes(), &definitionEnvelope); err != nil {
		t.Fatal(err)
	}
	if got := publicFieldKeys(definitionEnvelope.Data.Fields); !reflect.DeepEqual(got, []crud.FieldKey{"name", "state_name", "summary", "calculated_number"}) {
		t.Fatalf("read model fields = %#v", got)
	}
	for _, key := range []crud.FieldKey{"name", "state_name", "summary"} {
		if field := findPublicField(t, definitionEnvelope.Data.Fields, key); !field.ReadOnly {
			t.Fatalf("read model field %q is writable", key)
		}
	}
	if len(definitionEnvelope.Data.Form.Fields) != 0 || containsAction(definitionEnvelope.Data.Actions, crud.ActionCreate) || containsAction(definitionEnvelope.Data.Actions, crud.ActionUpdate) || containsAction(definitionEnvelope.Data.Actions, crud.ActionDelete) {
		t.Fatalf("read model exposes mutation metadata: form=%#v actions=%#v", definitionEnvelope.Data.Form.Fields, definitionEnvelope.Data.Actions)
	}

	recordsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoReadModelTable+"/records?page=1&size=10", nil)
	recordsResponse := httptest.NewRecorder()
	handler.ServeHTTP(recordsResponse, recordsRequest)
	if recordsResponse.Code != http.StatusOK {
		t.Fatalf("read model records = %d: %s", recordsResponse.Code, recordsResponse.Body.String())
	}
	var recordsEnvelope struct {
		Data []crud.Record `json:"data"`
	}
	if err := json.Unmarshal(recordsResponse.Body.Bytes(), &recordsEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(recordsEnvelope.Data) != 4 {
		t.Fatalf("read model records = %d, want 4", len(recordsEnvelope.Data))
	}
	if got := recordsEnvelope.Data[0].Fields["state_name"]; got != "Paraná" {
		t.Fatalf("joined state = %#v", got)
	}
	if got := recordsEnvelope.Data[0].Fields["summary"]; got != "Contato Curitiba — Paraná" {
		t.Fatalf("calculated summary = %#v", got)
	}
	if got := recordsEnvelope.Data[0].Fields["calculated_number"]; got != float64(2021) {
		t.Fatalf("calculated numeric value = %#v", got)
	}
	for _, record := range recordsEnvelope.Data {
		if record.Fields["state_name"] == "Real" || strings.Contains(record.Fields["summary"].(string), "Real") {
			t.Fatalf("read model leaked a non-state reference: %#v", record)
		}
	}

	writeRequest := httptest.NewRequest(http.MethodPut, "/api/v1/crud/"+demoReadModelTable+"/records/1", strings.NewReader(`{"version":1,"fields":{"summary":"tampered"}}`))
	writeRequest.Header.Set("Content-Type", "application/json")
	writeResponse := httptest.NewRecorder()
	handler.ServeHTTP(writeResponse, writeRequest)
	if writeResponse.Code != http.StatusForbidden {
		t.Fatalf("read model update = %d: %s", writeResponse.Code, writeResponse.Body.String())
	}
}

func TestDemoDefinitionsPreserveSchemaAndExplicitColumnOrder(t *testing.T) {
	database := newDemoDatabase(t)
	handler, err := newHandler(database)
	if err != nil {
		t.Fatal(err)
	}
	readDefinition := func(resource string) crud.PublicDefinition {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+resource+"/definition", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s definition = %d: %s", resource, response.Code, response.Body.String())
		}
		var envelope struct {
			Data crud.PublicDefinition `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode %s definition: %v", resource, err)
		}
		return envelope.Data
	}

	physical := readDefinition(demoTable)
	if got := publicFieldKeys(physical.Fields); !reflect.DeepEqual(got, []crud.FieldKey{"name", "notes", "status", "enabled"}) {
		t.Fatalf("automatic fields = %#v, want physical schema order", got)
	}
	if !reflect.DeepEqual(physical.Grid.Columns, []crud.FieldKey{"name", "notes", "status", "enabled"}) {
		t.Fatalf("automatic grid columns = %#v, want physical schema order", physical.Grid.Columns)
	}

	explicit := readDefinition(demoOrderedTable)
	if got := publicFieldKeys(explicit.Fields); !reflect.DeepEqual(got, []crud.FieldKey{"name", "notes", "status", "enabled"}) {
		t.Fatalf("explicit fields = %#v, want canonical physical order", got)
	}
	wantProjection := []crud.FieldKey{"status", "name", "enabled", "notes"}
	if !reflect.DeepEqual(explicit.Grid.Columns, wantProjection) || !reflect.DeepEqual(explicit.Form.Fields, wantProjection) {
		t.Fatalf("explicit projections = grid %#v form %#v, want %#v", explicit.Grid.Columns, explicit.Form.Fields, wantProjection)
	}
}

func TestDemoArchivedVisibilityListsArchivedRowsAsReadOnlyMetadata(t *testing.T) {
	database := newDemoDatabase(t)
	handler, err := newHandler(database)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoEnumTable+"/records", strings.NewReader(`{"fields":{"name":"Archive visibility smoke","status":"new","enabled":true,"notes":"archive me"}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", response.Code, response.Body.String())
	}
	var created struct {
		Data crud.Record `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodDelete, "/api/v1/crud/"+demoEnumTable+"/records/"+string(created.Data.ID), strings.NewReader(`{"version":1,"fields":{}}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("archive = %d: %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoEnumTable+"/records?include_archived=true", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("include archived = %d: %s", response.Code, response.Body.String())
	}
	var listed struct {
		Data []crud.Record `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	for _, record := range listed.Data {
		if record.ID == created.Data.ID {
			if !record.Archived {
				t.Fatalf("archived record metadata = %#v", record)
			}
			return
		}
	}
	t.Fatalf("archived record %q was not returned", created.Data.ID)
}

func TestDemoStaticDefaultsArePublishedAndAppliedOnCreate(t *testing.T) {
	database := newDemoDatabase(t)
	handler, err := newHandler(database)
	if err != nil {
		t.Fatal(err)
	}
	definitionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/crud/"+demoDefaultsTable+"/definition", nil)
	definitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(definitionResponse, definitionRequest)
	if definitionResponse.Code != http.StatusOK {
		t.Fatalf("defaults definition = %d: %s", definitionResponse.Code, definitionResponse.Body.String())
	}
	var definitionEnvelope struct {
		Data crud.PublicDefinition `json:"data"`
	}
	if err := json.Unmarshal(definitionResponse.Body.Bytes(), &definitionEnvelope); err != nil {
		t.Fatal(err)
	}
	if status := findPublicField(t, definitionEnvelope.Data.Fields, "status"); status.Default != "new" {
		t.Fatalf("status default = %#v", status.Default)
	}
	if enabled := findPublicField(t, definitionEnvelope.Data.Fields, "enabled"); enabled.Default != true {
		t.Fatalf("enabled default = %#v", enabled.Default)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/crud/"+demoDefaultsTable+"/records", strings.NewReader(`{"fields":{"name":"Defaulted record","notes":"server default smoke"}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("defaults create = %d: %s", response.Code, response.Body.String())
	}
	var recordEnvelope struct {
		Data crud.Record `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &recordEnvelope); err != nil {
		t.Fatal(err)
	}
	if recordEnvelope.Data.Fields["status"] != "new" || recordEnvelope.Data.Fields["enabled"] != float64(1) {
		t.Fatalf("created default fields = %#v", recordEnvelope.Data.Fields)
	}
}

func TestDemoPortsAreStableAndExplicit(t *testing.T) {
	principal, err := (demoPrincipal{}).Principal(context.Background())
	if err != nil || principal.ID != "disposable-demo-operator" {
		t.Fatalf("principal = %#v, %v", principal, err)
	}
	scope, err := (demoScope{}).Scope(context.Background(), demoTable)
	if err != nil || scope["tenant_id"] != "demo_tenant_alpha" {
		t.Fatalf("scope = %#v, %v", scope, err)
	}
	if err := (demoAuthorizer{}).Authorize(context.Background(), principal, demoTable, crud.ActionRead, nil); err != nil {
		t.Fatal(err)
	}
	if err := (demoAudit{}).Append(context.Background(), crud.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	if got := (demoTranslator{}).Message(context.Background(), "demo.message", nil); got != "demo.message" {
		t.Fatalf("message = %q", got)
	}
	if (demoClock{}).Now().After(time.Now().UTC().Add(time.Second)) {
		t.Fatal("clock returned a future time")
	}
}

func TestDemoAddressUsesTheConfiguredEnvironmentOrDefault(t *testing.T) {
	t.Setenv("CLEAR_CRUD_DEMO_ADDR", "127.0.0.1:9099")
	if got := demoAddress(); got != "127.0.0.1:9099" {
		t.Fatalf("configured demo address = %q", got)
	}
	t.Setenv("CLEAR_CRUD_DEMO_ADDR", "")
	if got := demoAddress(); got != "127.0.0.1:8088" {
		t.Fatalf("default demo address = %q", got)
	}
}

func TestRunBuildsAndServesTheDemo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.sqlite")
	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoMinimalSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoIndex); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoMinimalIndex); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoEnumControlsSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoEnumControlsIndex); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoReferenceSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoReferenceIndexes); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	want := errors.New("listener stopped")
	err = run(path, "127.0.0.1:0", func(address string, handler http.Handler) error {
		if address != "127.0.0.1:0" {
			t.Fatalf("address = %q", address)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("healthz = %d", response.Code)
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("run error = %v", err)
	}
	if err := run("file:/not/a/demo.sqlite?mode=ro", "127.0.0.1:0", func(string, http.Handler) error { return nil }); err == nil {
		t.Fatal("missing database accepted")
	}
}

func newDemoDatabase(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(demoSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoMinimalSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoIndex); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoMinimalIndex); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoEnumControlsSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoEnumControlsIndex); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoReferenceSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(demoReferenceIndexes); err != nil {
		t.Fatal(err)
	}
	return database
}

func findPublicField(t *testing.T, fields []crud.Field, key crud.FieldKey) crud.Field {
	t.Helper()
	for _, field := range fields {
		if field.Key == key {
			return field
		}
	}
	t.Fatalf("public field %q absent", key)
	return crud.Field{}
}

func publicFieldKeys(fields []crud.Field) []crud.FieldKey {
	keys := make([]crud.FieldKey, len(fields))
	for index, field := range fields {
		keys[index] = field.Key
	}
	return keys
}

func containsAction(actions []crud.Action, wanted crud.Action) bool {
	for _, action := range actions {
		if action == wanted {
			return true
		}
	}
	return false
}

const demoSchema = `CREATE TABLE clear_crud_disposable_junk_contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT NOT NULL,
    name VARCHAR(120) NOT NULL,
    notes TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'new',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    version INTEGER NOT NULL DEFAULT 1,
    archived INTEGER NOT NULL DEFAULT 0,
    zextra_01 TEXT,
    zextra_02 TEXT,
    zextra_03 TEXT,
    zextra_04 TEXT,
    zextra_05 TEXT,
    zextra_06 TEXT,
    zextra_07 TEXT,
    zextra_08 TEXT,
    zextra_09 TEXT,
    zextra_10 TEXT,
    zextra_11 TEXT,
    zextra_12 TEXT,
    zextra_13 TEXT
)`

const demoMinimalSchema = `CREATE TABLE clear_crud_minimal_demo_contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    notes TEXT,
    status TEXT NOT NULL DEFAULT 'new',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    version INTEGER NOT NULL DEFAULT 1
)`

const demoIndex = `CREATE INDEX idx_clear_crud_disposable_junk_contacts_listing ON clear_crud_disposable_junk_contacts (tenant_id, enabled, name)`

const demoMinimalIndex = `CREATE INDEX idx_clear_crud_minimal_demo_contacts_listing ON clear_crud_minimal_demo_contacts (tenant_id, enabled, name)`

const demoEnumControlsSchema = `CREATE TABLE clear_crud_enum_controls_demo (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    priority TEXT NOT NULL CHECK (priority IN ('low', 'medium', 'high')),
    channel TEXT NOT NULL CHECK (channel IN ('email', 'phone', 'whatsapp')),
    state TEXT NOT NULL CHECK (state IN ('draft', 'active', 'closed')),
    tone TEXT NOT NULL CHECK (tone IN ('info', 'warning', 'urgent')),
    version INTEGER NOT NULL DEFAULT 1
)`

const demoEnumControlsIndex = `CREATE INDEX idx_clear_crud_enum_controls_demo_listing ON clear_crud_enum_controls_demo (tenant_id, name)`

const demoReferenceSchema = `CREATE TABLE clear_crud_reference_values (id INTEGER PRIMARY KEY, kind TEXT NOT NULL, name TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1);
CREATE TABLE clear_crud_reference_regions (id INTEGER PRIMARY KEY, state_id INTEGER NOT NULL, name TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1);
CREATE TABLE clear_crud_reference_cities (id INTEGER PRIMARY KEY, region_id INTEGER NOT NULL, name TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1);
CREATE TABLE clear_crud_reference_lookup_demo (id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id TEXT NOT NULL, name TEXT NOT NULL, state_id INTEGER NOT NULL, region_id INTEGER NOT NULL, city_id INTEGER NOT NULL, version INTEGER NOT NULL DEFAULT 1);
CREATE TABLE clear_crud_auxiliary_catalogs (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, code INTEGER NOT NULL, title TEXT NOT NULL, max_options INTEGER NOT NULL, active BOOLEAN NOT NULL DEFAULT 1, management TEXT NOT NULL, value_1_label TEXT NOT NULL, value_1_type TEXT NOT NULL, value_1_required BOOLEAN NOT NULL, value_2_label TEXT NOT NULL, value_2_type TEXT NOT NULL, value_2_required BOOLEAN NOT NULL, version INTEGER NOT NULL DEFAULT 1);
CREATE TABLE clear_crud_auxiliary_catalog_options (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, catalog_id TEXT NOT NULL REFERENCES clear_crud_auxiliary_catalogs(id), sequence INTEGER NOT NULL, value_1 TEXT NOT NULL, value_2 TEXT NOT NULL, active BOOLEAN NOT NULL DEFAULT 1, sort_order INTEGER NOT NULL, version INTEGER NOT NULL DEFAULT 1, UNIQUE (catalog_id, sequence));
CREATE TABLE clear_crud_auxiliary_catalog_lookup_demo (id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id TEXT NOT NULL, name TEXT NOT NULL, selected_option TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1);
INSERT INTO clear_crud_reference_values (id, kind, name) VALUES
 (1, 'state', 'São Paulo'), (2, 'state', 'Paraná'), (3, 'state', 'Acre'), (4, 'state', 'Alagoas'), (5, 'state', 'Amapá'),
 (6, 'state', 'Amazonas'), (7, 'state', 'Bahia'), (8, 'state', 'Ceará'), (9, 'state', 'Distrito Federal'), (10, 'state', 'Espírito Santo'),
 (11, 'state', 'Goiás'), (12, 'state', 'Maranhão'), (13, 'state', 'Mato Grosso'), (14, 'state', 'Mato Grosso do Sul'), (15, 'state', 'Minas Gerais'),
 (16, 'state', 'Pará'), (17, 'state', 'Paraíba'), (18, 'state', 'Pernambuco'), (19, 'state', 'Piauí'), (20, 'state', 'Rio de Janeiro'),
 (21, 'state', 'Rio Grande do Norte'), (22, 'state', 'Rio Grande do Sul'), (23, 'state', 'Rondônia'), (24, 'state', 'Roraima'),
 (25, 'state', 'Santa Catarina'), (26, 'state', 'Sergipe'), (27, 'state', 'Tocantins'), (101, 'currency', 'Real');
INSERT INTO clear_crud_reference_regions (id, state_id, name) VALUES (11, 1, 'Região Metropolitana de São Paulo'), (12, 1, 'Vale do Paraíba'), (21, 2, 'Região Metropolitana de Curitiba');
INSERT INTO clear_crud_reference_cities (id, region_id, name) VALUES (111, 11, 'São Paulo'), (112, 11, 'Osasco'), (121, 12, 'Taubaté'), (211, 21, 'Curitiba');
INSERT INTO clear_crud_reference_lookup_demo (tenant_id, name, state_id, region_id, city_id) VALUES ('demo_tenant_alpha', 'Contato São Paulo', 1, 11, 111), ('demo_tenant_alpha', 'Contato Osasco', 1, 11, 112), ('demo_tenant_alpha', 'Contato Taubaté', 1, 12, 121), ('demo_tenant_alpha', 'Contato Curitiba', 2, 21, 211);
INSERT INTO clear_crud_auxiliary_catalogs (id, tenant_id, code, title, max_options, management, value_1_label, value_1_type, value_1_required, value_2_label, value_2_type, value_2_required) VALUES (10, 'demo_tenant_alpha', 10, 'Estados', 99, 'fixed', 'Sigla', 'text', 1, 'Nome', 'text', 0), (20, 'demo_tenant_alpha', 20, 'Territórios', 99, 'fixed', 'Código', 'text', 1, 'Nome', 'text', 0), (30, 'demo_tenant_alpha', 30, 'Canais', 20, 'fixed', 'Código', 'text', 1, 'Nome', 'text', 0), (40, 'demo_tenant_alpha', 40, 'Prioridades', 10, 'fixed', 'Código', 'text', 1, 'Nome', 'text', 0);
INSERT INTO clear_crud_auxiliary_catalog_options (id, tenant_id, catalog_id, sequence, value_1, value_2, active, sort_order) VALUES (1001, 'demo_tenant_alpha', 10, 1, 'SP', 'São Paulo', 1, 10), (1002, 'demo_tenant_alpha', 10, 2, 'PR', 'Paraná', 1, 20), (2001, 'demo_tenant_alpha', 20, 1, 'DF', 'Distrito Federal', 1, 10), (2002, 'demo_tenant_alpha', 20, 2, 'FN', 'Fernando de Noronha', 1, 20), (2003, 'demo_tenant_alpha', 20, 3, 'OLD', 'Território inativo', 0, 30), (3001, 'demo_tenant_alpha', 30, 1, 'email', 'E-mail', 1, 10), (3002, 'demo_tenant_alpha', 30, 2, 'phone', 'Telefone', 1, 20), (3003, 'demo_tenant_alpha', 30, 3, 'whatsapp', 'WhatsApp', 1, 30), (3004, 'demo_tenant_alpha', 30, 4, 'FAX', 'Canal inativo', 0, 40), (4001, 'demo_tenant_alpha', 40, 1, 'LOW', 'Baixa', 1, 10), (4002, 'demo_tenant_alpha', 40, 2, 'HIGH', 'Alta', 1, 20);
INSERT INTO clear_crud_auxiliary_catalog_lookup_demo (tenant_id, name, selected_option) VALUES ('demo_tenant_alpha', 'Seleção de território', 'DF'), ('demo_tenant_alpha', 'Seleção de estado', 'SP'), ('demo_tenant_alpha', 'Seleção de estado alternativo', 'PR')`

const demoReferenceIndexes = `CREATE INDEX idx_clear_crud_reference_values_kind_name ON clear_crud_reference_values (kind, name);
CREATE INDEX idx_clear_crud_reference_regions_state_name ON clear_crud_reference_regions (state_id, name);
CREATE INDEX idx_clear_crud_reference_cities_region_name ON clear_crud_reference_cities (region_id, name);
CREATE INDEX idx_clear_crud_reference_lookup_demo_listing ON clear_crud_reference_lookup_demo (tenant_id, name);
CREATE INDEX idx_clear_crud_auxiliary_catalogs_listing ON clear_crud_auxiliary_catalogs (tenant_id, active, title);
CREATE INDEX idx_clear_crud_auxiliary_catalog_options_lookup ON clear_crud_auxiliary_catalog_options (tenant_id, catalog_id, active, sort_order, value_1);
CREATE INDEX idx_clear_crud_auxiliary_catalog_options_listing ON clear_crud_auxiliary_catalog_options (tenant_id, catalog_id, sort_order);
CREATE INDEX idx_clear_crud_auxiliary_catalog_lookup_demo_listing ON clear_crud_auxiliary_catalog_lookup_demo (tenant_id, name)`
