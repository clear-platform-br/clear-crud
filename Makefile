.PHONY: validate test coverage

COVERAGE_MIN ?= 90
COVERAGE_FILE := /private/tmp/clear-crud-coverage.out

validate: test coverage

test:
	GOCACHE=/private/tmp/clear-crud-gocache go build ./...
	GOCACHE=/private/tmp/clear-crud-gocache go test ./...
	GOCACHE=/private/tmp/clear-crud-gocache go test -race ./...
	GOCACHE=/private/tmp/clear-crud-gocache go vet ./...

coverage:
	GOCACHE=/private/tmp/clear-crud-gocache go test -coverprofile=$(COVERAGE_FILE) ./...
	@total=$$(GOCACHE=/private/tmp/clear-crud-gocache go tool cover -func=$(COVERAGE_FILE) | awk '/^total:/{gsub("%", "", $$3); print $$3}'); \
	awk -v total="$$total" -v minimum="$(COVERAGE_MIN)" 'BEGIN { if (total + 0 < minimum + 0) { printf "coverage %.1f%% is below required %.1f%%\n", total, minimum; exit 1 } printf "coverage %.1f%% meets required %.1f%%\n", total, minimum }'
