SHELL := bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:        # use a single shell for commands instead a new shell per line
.DELETE_ON_ERROR: # delete output files when make rule fails
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

version := $(shell date '+%Y-%m-%d')
commit := $(shell git rev-parse --short HEAD)
ldflags := -ldflags "-X 'main.version=${version}' -X 'main.commit=${commit}'"

.PHONY: all
all: lint test acceptance-test

.PHONY: start
start:
	docker-compose up -d

.PHONY: stop
stop:
	docker-compose down

.PHONY: restart
restart: stop start

.PHONY: psql
psql:
	PGPASSWORD=hunter2 psql --host=127.0.0.1 --port=5555 --username=postgres pggen

.PHONY: test
test:
	go test ./...

.PHONY: acceptance-test
acceptance-test:
	DOCKER_API_VERSION=1.39 go test ./example/acceptance_test.go

.PHONY: update-acceptance-test
update-acceptance-test:
	go test ./example/acceptance_test.go -update

.PHONY: lint
lint:
	golangci-lint run

.PHONY: binary-all
binary-all: binary-darwin-amd64 binary-darwin-arm64 binary-linux-amd64 binary-windows-amd64

.PHONY: dist-dir
dist-dir:
	mkdir -p dist

.PHONY: binary-darwin-amd64
binary-darwin-amd64: dist-dir
	GOOS=darwin GOARCH=amd64 go build ${ldflags} -o dist/pggen-darwin-amd64 ./cmd/pggen

.PHONY: binary-darwin-arm64
binary-darwin-arm64: dist-dir
	GOOS=darwin GOARCH=arm64 go build ${ldflags} -o dist/pggen-darwin-arm64 ./cmd/pggen

.PHONY: binary-linux-amd64
binary-linux-amd64: dist-dir
	GOOS=linux GOARCH=amd64 go build ${ldflags} -o dist/pggen-linux-amd64 ./cmd/pggen

.PHONY: binary-windows-amd64
binary-windows-amd64: dist-dir
	GOOS=windows GOARCH=amd64 go build ${ldflags} -o dist/pggen-windows-amd64.exe ./cmd/pggen

.PHONY: release
release: binary-all
	./script/release.sh
