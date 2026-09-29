.PHONY: validate test coverage security frontend

COVERAGE_MIN ?= 90
CLEAR_CRUD_CACHE_ROOT ?= $(if $(TMPDIR),$(TMPDIR),/tmp)/clear-crud
CLEAR_CRUD_GOCACHE ?= $(CLEAR_CRUD_CACHE_ROOT)/go-build
COVERAGE_FILE := $(CLEAR_CRUD_CACHE_ROOT)/coverage.out
GOVULNCHECK ?= $(shell command -v govulncheck 2>/dev/null || printf '%s/bin/govulncheck' "$$(go env GOPATH)")

validate: test coverage security frontend

test:
	GOCACHE="$(CLEAR_CRUD_GOCACHE)" go build ./...
	GOCACHE="$(CLEAR_CRUD_GOCACHE)" go test ./...
	GOCACHE="$(CLEAR_CRUD_GOCACHE)" go test -race ./...
	GOCACHE="$(CLEAR_CRUD_GOCACHE)" go vet ./...

coverage:
	GOCACHE="$(CLEAR_CRUD_GOCACHE)" go test -coverprofile=$(COVERAGE_FILE) ./...
	@total=$$(GOCACHE="$(CLEAR_CRUD_GOCACHE)" go tool cover -func=$(COVERAGE_FILE) | awk '/^total:/{gsub("%", "", $$3); print $$3}'); \
	awk -v total="$$total" -v minimum="$(COVERAGE_MIN)" 'BEGIN { if (total + 0 < minimum + 0) { printf "coverage %.1f%% is below required %.1f%%\n", total, minimum; exit 1 } printf "coverage %.1f%% meets required %.1f%%\n", total, minimum }'

security:
	@test -x "$(GOVULNCHECK)" || { echo "govulncheck is required; install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	GOCACHE="$(CLEAR_CRUD_GOCACHE)" "$(GOVULNCHECK)" ./...

frontend:
	npm_config_cache="$(CURDIR)/work/npm-cache" npm ci
	npm_config_cache="$(CURDIR)/work/npm-cache" npm run frontend:build
	npm_config_cache="$(CURDIR)/work/npm-cache" npm run frontend:test
	npm_config_cache="$(CURDIR)/work/npm-cache" npm run frontend:pack
