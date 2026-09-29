// Command clear-crud-demo serves the disposable renderer preview locally.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	crud "github.com/clear-platform-br/clear-crud"
	"github.com/clear-platform-br/clear-crud/httpadapter"
	_ "modernc.org/sqlite"
)

const demoDatabaseDSN = "file:demo/clear_crud_disposable_junk.sqlite?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"

func main() {
	if err := run(demoDatabaseDSN, demoAddress(), http.ListenAndServe); err != nil {
		log.Fatal(err)
	}
}

func run(databaseDSN, address string, serve func(string, http.Handler) error) error {
	database, err := sql.Open("sqlite", databaseDSN)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		return err
	}
	handler, err := newHandler(database)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/crud/", handler)
	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusNoContent) })
	log.Printf("clear-crud disposable demo API listening on http://%s", address)
	return serve(address, mux)
}

func demoAddress() string {
	if address := os.Getenv("CLEAR_CRUD_DEMO_ADDR"); address != "" {
		return address
	}
	return "127.0.0.1:8088"
}

func newHandler(database *sql.DB) (http.Handler, error) {
	registry := crud.NewRegistry()
	if err := registerDefinitions(context.Background(), database, registry); err != nil {
		return nil, err
	}
	registry.Seal()
	service, err := crud.NewService(crud.Dependencies{
		Registry: registry, Principal: demoPrincipal{}, Scope: demoScope{}, Authorizer: demoAuthorizer{}, Audit: demoAudit{}, Translator: demoTranslator{}, Clock: demoClock{},
	})
	if err != nil {
		return nil, err
	}
	return httpadapter.New(service, httpadapter.Options{Translator: demoTranslator{}})
}

type demoPrincipal struct{}

func (demoPrincipal) Principal(context.Context) (crud.Principal, error) {
	return crud.Principal{ID: "disposable-demo-operator", Kind: "demo"}, nil
}

type demoScope struct{}

func (demoScope) Scope(context.Context, crud.ResourceKey) (crud.Scope, error) {
	return crud.Scope{"tenant_id": "demo_tenant_alpha"}, nil
}

type demoAuthorizer struct{}

func (demoAuthorizer) Authorize(context.Context, crud.Principal, crud.ResourceKey, crud.Action, *crud.Record) error {
	return nil
}

type demoAudit struct{}

func (demoAudit) Append(context.Context, crud.AuditEvent) error { return nil }

type demoTranslator struct{}

func (demoTranslator) Message(_ context.Context, code crud.MessageCode, _ map[string]any) string {
	return string(code)
}

type demoClock struct{}

func (demoClock) Now() time.Time { return time.Now().UTC() }
