SHELL := bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:        # use a single shell for commands instead a new shell per line
.DELETE_ON_ERROR: # delete output files when make rule fails
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

.PHONY: start
start:
	docker-compose up -d

.PHONY: stop
stop:
	docker-compose down

.PHONY: statik
statik:
	statik -m -dest=internal -src=internal/codegen -include='*.gotemplate'
	gofmt -w internal/statik/statik.go

.PHONY: psql
psql:
	PGPASSWORD=hunter2 psql --host=127.0.0.1 --port=5555 --username=postgres pggen

.PHONY: test
test:
	go test ./...

.PHONY: integration-test
integration-test:
	go test ./example/integration_test.go

.PHONY: update-integration-test
update-integration-test:
	go test ./example/integration_test.go -update

.PHONY: lint
lint:
	golangci-lint run
